package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/pterm/pterm"
	"golang.org/x/exp/slices"
)

var (
	promptHistoryFile      = fmt.Sprintf("%s/.config/laizy/.prompt_history", userHomeFolder)
	bannerQOTD             string
	clear                  map[string]func() //create a map for storing clear funcs
	userHomeFolder         = os.Getenv("HOME")
	laizyAPIKey            = os.Getenv("LAIZY_API_KEY")
	promptHistory          = []string{}
	promptValue            string
	lastPrompt             string
	statusIcon             string
	laizyInputMultiLine    = false
	laizyInputChain        = false
	laizyInputFile         = ""
	laizyInputFileContents = []byte{}
	laizyFullResponse      = ""
	laizyLastResponse      = ""
	laizyQOTDMessages      = []string{
		"You are a star!",
		"Keep up the good work!",
		"Your hard work will pay off!",
		"You are doing great!",
		"You are a rockstar!",
		"You are a genius!",
		"You are a superstar!",
		"You are a legend!",
		"You are a champion!",
	}
	helpMenuEntries = []string{
		"Laizy CLI",
		"Type exit or quit to exit",
		"Type clear to clear the screen",
		"Type help to show this menu",
		"Type %load (%ld) to load data from a file",
		"Type %save to save the current output to a file",
		"Type %exec (%execs) to execute a shell command, add s to save as data",
		"Type %history to show the command history",
		"Type %hl <#> to load a command from the history",
		"Type %multi to toggle multiline mode",
		"Type %chain to toggle chaining (prompt-output-prompt) mode",
		"Type %fetch to fetch data from a url",
		"Type %tips to show a random tip",
	}
)

func CallClear() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}
func init() {

	clear = make(map[string]func())
	clear["linux"] = func() {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["darwin"] = func() {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	rand.Seed(time.Now().Unix())
	randomBannerMessage := laizyQOTDMessages[rand.Intn(len(laizyQOTDMessages))]
	bannerQOTD = pterm.NewStyle(pterm.FgLightMagenta).Sprint(randomBannerMessage)

	// attempt to create laizy config folder
	laizyConfigFolder := fmt.Sprintf("%s/.config/laizy", userHomeFolder)
	_, err := os.Stat(laizyConfigFolder)
	if os.IsNotExist(err) {
		os.Mkdir(laizyConfigFolder, 0700)
	}
	promptHistoryFile, err := os.ReadFile(promptHistoryFile)
	if err != nil {
		pterm.Error.Println(err)
	}
	promptHistory = strings.Split(string(promptHistoryFile), "\n")
}

func specialCommandHandler(userPrompt string) bool {
	var err error
	unmodifiedPrompt := userPrompt
	userPrompt = strings.Split(userPrompt, " ")[0]
	promptHistory = append(promptHistory, unmodifiedPrompt)
	switch userPrompt {
	case "%inspect":
		pterm.Info.Println("Inspecting laizy's short term memory")
		pterm.Info.Println("Last Response:", laizyLastResponse)
		pterm.Info.Println("Full Response:", laizyFullResponse)
		pterm.Info.Println("Last Prompt:", lastPrompt)
		pterm.Info.Println("Prompt Value:", promptValue)
		pterm.Info.Println("Input MultiLine:", laizyInputMultiLine)
		pterm.Info.Println("Input Chain:", laizyInputChain)

		return true
	case "%forget":
		// clear laizy short term memory
		laizyLastResponse = ""
		laizyFullResponse = ""
		lastPrompt = ""
		pterm.Success.Println("Laizy short-term memory cleared")
		return true
	case "%multi":
		laizyInputMultiLine = !laizyInputMultiLine
		if laizyInputMultiLine {
			pterm.Info.Println("Multi line input enabled")
		} else {
			pterm.Info.Println("Single line input enabled")
		}

		return true
	case "%chain":
		laizyInputChain = !laizyInputChain
		if laizyInputChain {
			pterm.Info.Println("Chain input enabled")
		} else {
			pterm.Info.Println("Chain input disabled")
		}

		return true

	case "%clear":
		CallClear()
		return true
	case "%exit", "%quit", "exit", "quit":
		os.Exit(0)
		return true
	case "%help", "help":

		for _, entry := range helpMenuEntries {
			pterm.Info.Println(entry)
		}

		return true
	case "%history":
		pterm.Info.Println("Laizy CLI Prompt History")
		for index, prompt := range promptHistory {
			pterm.Info.Println(index, prompt)
		}
		return true
	case "%hl":
		// load from history - clear the existing prompt and load the selected prompt
		laizyLastResponse = ""
		laizyFullResponse = ""
		lastPrompt = ""
		promptValue = ""
		// disable chain mode
		laizyInputChain = false
		var historyItem int
		if len(strings.Split(unmodifiedPrompt, " ")) == 1 {
			// use last prompt
			historyItem = len(promptHistory) - 2
			// history item
		} else {
			historyItem, err = strconv.Atoi(strings.Split(unmodifiedPrompt, " ")[1])
		}
		if historyItem > len(promptHistory) {
			pterm.Error.Println("Line number out of range")
			return true
		}

		// ignore commands with % prefix
		isCommand := strings.HasPrefix(promptHistory[historyItem], "%")
		if isCommand {
			pterm.Error.Println("Cannot load command from history")
			return true
		}

		if err != nil {
			pterm.Error.Println(err)
			return true
		}
		promptValue = promptHistory[historyItem]
		// print loaded prompt
		pterm.Info.Println("Loaded prompt from history: ", promptValue)
		// press enter to
		pterm.Info.Println("Press enter to generate output from this prompt")
		return true

	case "%ld", "%load":
		if len(strings.Split(unmodifiedPrompt, " ")) > 1 {
			laizyInputFile = strings.Split(unmodifiedPrompt, " ")[1]
			laizyInputFile = strings.TrimSpace(laizyInputFile)
		} else {
			laizyInputFile, _ = pterm.DefaultInteractiveTextInput.Show("Enter a filename to load the data from")
		}
		laizyInputFileContents, err = os.ReadFile(laizyInputFile)
		if err != nil {
			pterm.Error.Println("Error loading file", err)
			return true
		}

		laizyLastResponse = string(laizyInputFileContents)
		lastPrompt = laizyLastResponse
		laizyFullResponse = ""
		pterm.Println("loaded data from file\n", lastPrompt)
		return true
	case "%exec", "%execs", "%!":
		if len(strings.Split(unmodifiedPrompt, " ")) > 1 {
			baseCommand := strings.Split(unmodifiedPrompt, " ")[1]
			shellCommandWithArgs := strings.Split(unmodifiedPrompt, " ")[2:]
			out, err := exec.Command(baseCommand, shellCommandWithArgs...).Output()
			if err != nil {
				pterm.Error.Println(err)
			}
			pterm.Println(string(out))
			if userPrompt == "%execs" {
				laizyLastResponse = string(out)
			}
		} else {
			pterm.Error.Println("No shell command provided")
		}

		return true
	case "%fetch", "%curl":
		if len(strings.Split(unmodifiedPrompt, " ")) > 1 {
			url := strings.Split(unmodifiedPrompt, " ")[1]
			// check for http/s
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = fmt.Sprintf("http://%s", url)
			}
			resp, err := http.DefaultClient.Get(url)
			if err != nil {
				pterm.Error.Println(err)
				return true
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				pterm.Error.Println(err)
				return true
			}
			pterm.Println(string(body))
			laizyLastResponse = string(body)

		} else {
			pterm.Error.Println("No URL provided")
		}

		return true
	case "%save", "%s":
		var laizyOutputFile string
		if len(strings.Split(unmodifiedPrompt, " ")) > 1 {
			laizyOutputFile = strings.Split(unmodifiedPrompt, " ")[1]
		} else {
			laizyOutputFile, _ = pterm.DefaultInteractiveTextInput.Show("Enter a filename to save the last response to")
		}
		pterm.Info.Println("Saving prompt output to file", laizyOutputFile)
		f, err := os.OpenFile(laizyOutputFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			pterm.Error.Println("Error creating file", err, laizyOutputFile)
		}
		defer f.Close()
		_, err = io.WriteString(f, laizyFullResponse)
		if err != nil {
			pterm.Error.Println("Error writing to file", laizyOutputFile)
		}
		return true
	case "%tip", "%tips":
		exampleText := promptSuggestions[rand.Intn(len(promptSuggestions))]
		pterm.Info.Println(exampleText)
		return true

	}

	if strings.HasPrefix(userPrompt, "%") {
		pterm.Error.Println("Unknown command", userPrompt)
		return true
	}
	return false
}

func main() {

	// clear screen on linux
	CallClear()
	if laizyAPIKey == "" {
		pterm.Error.Println("Laizy API Key not set - please set LAIZY_API_KEY environment variable")
		pterm.Error.Println("You can get a free API key at https://app.laizy.dev")
		os.Exit(1)
	}
	// show laizy.dev header
	pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).WithTextStyle(pterm.NewStyle(pterm.FgLightYellow)).WithMargin(5).Println("Laizy AI")
	pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).WithMargin(5).Println(bannerQOTD)
	// show laiz.dev logo
	pterm.DefaultCenter.Println(laizyLogo)
	// print sample prompts for users
	pterm.DefaultCenter.Println(pterm.NewStyle(pterm.FgLightMagenta).Sprint("Tips & Tricks:"))
	// choose 3 random prompts from the list
	rand.Seed(time.Now().UnixNano())
	var randomSuggestions []string

	for i := 0; i < 3; i++ {
		// prevent duplicate examples
		exampleText := promptSuggestions[rand.Intn(len(promptSuggestions))]
		if slices.Contains(randomSuggestions, exampleText) {
			i--
			continue
		}

		pterm.DefaultCenter.Println(pterm.NewStyle(pterm.FgLightMagenta).Sprint(exampleText))
	}

	for {
		var laizySpinnerMessage = "thinking"
		var laizyResponseJSON map[string]interface{}
		inputPromptStyle := pterm.NewStyle(pterm.FgLightYellow, pterm.BgLightBlue)
		var userPromptValue string
		if laizyInputChain {
			statusIcon = "⛓"
		} else {
			statusIcon = ""
		}
		if laizyInputFile != "" {
			if len(laizyInputFileContents) > 0 {
				statusIcon = "💾"
			}
			laizyInputFileContents = nil
			laizyInputFile = ""
			// fmt.Println(len(laizyInputFileContents))
		}
		laizyPrompt := fmt.Sprintf("%sLAIZY>", statusIcon)
		if laizyInputMultiLine {
			userPromptValue, _ = pterm.DefaultInteractiveTextInput.WithMultiLine().WithTextStyle(inputPromptStyle).Show(laizyPrompt)
		} else {
			userPromptValue, _ = pterm.DefaultInteractiveTextInput.WithTextStyle(inputPromptStyle).Show(laizyPrompt)
		}
		if strings.Split(userPromptValue, " ")[0] == "" {
			// drop the empty index
			userPromptValue = strings.Join(strings.Split(userPromptValue, " ")[1:], " ")
		}

		if specialCommandHandler(userPromptValue) {
			continue
		}

		// if user prompt is blank treat it as a continuation of the prompt + response
		if len(userPromptValue) == 0 {
			if len(laizyLastResponse) != 0 {
				// continue from last prompt
				promptValue = lastPrompt + laizyFullResponse
				laizySpinnerMessage = "continuing from last response"
				// clear screen

			}

		} else {
			promptValue = userPromptValue

			lastPrompt = userPromptValue
			laizyFullResponse = ""
			if userPromptValue != "" {
				promptHistory = append(promptHistory, userPromptValue)
			}
			f, err := os.Create(promptHistoryFile)
			if err != nil {
				pterm.Error.Println("Error creating file", promptHistoryFile)
			}
			_, err = io.WriteString(f, strings.Join(promptHistory, "\n"))
			if err != nil {
				pterm.Error.Println("Error updating prompt history file", promptHistoryFile)
			}

		}
		if laizyInputChain {
			previousPrompt := promptHistory[len(promptHistory)-1]
			promptValue = previousPrompt + "\n" + laizyLastResponse + "\n" + userPromptValue
		}
		spinnerInfo, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Laizy is %s...", laizySpinnerMessage))
		laizyResponse := sendLaizyRequest(promptValue, 1)
		err := json.Unmarshal([]byte(laizyResponse), &laizyResponseJSON)
		if err != nil {
			spinnerInfo.Fail("An error occured")
		} else {
			spinnerInfo.Success()
			if laizyResponseJSON["content"] != nil {
				laizyLastResponse = laizyResponseJSON["content"].(string)
				laizyFullResponse = laizyFullResponse + laizyLastResponse
				pterm.Println(pterm.NewStyle(pterm.FgLightCyan).Sprint(laizyFullResponse)) // main loop
			} else {
				pterm.Error.Println("Laizy was unable to process your request")
			}
		}
	}
}

func sendLaizyRequest(userPrompt string, iterations int) string {
	url := "https://app.laizy.dev/submit"
	laizyRequestBody := gabs.New()
	laizyRequestBody.Set(userPrompt, "prompt")
	laizyRequestBody.Set(iterations, "iterations")
	laizyReader := strings.NewReader(laizyRequestBody.String())

	req, err := http.NewRequest("POST", url, laizyReader)
	if err != nil {
		pterm.Error.Println(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", laizyAPIKey)
	req.Header.Set("User-Agent", "laizy-cli")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		pterm.Error.Println("Error accessing laizy api", err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		pterm.Error.Println("Error processing laizy response", err)
	}
	return (string(body))
}
