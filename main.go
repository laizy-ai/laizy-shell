package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/pterm/pterm"
)

var (
	laizyAPIKey            = os.Getenv("LAIZY_API_KEY")
	promptHistory          = []string{}
	userHomeFolder         = os.Getenv("HOME")
	promptHistoryFile      = fmt.Sprintf("%s/.config/laizy/.prompt_history", userHomeFolder)
	promptValue            = ""
	lastPrompt             = ""
	laizyInputMultiLine    = false
	laizyInputChain   = false
	laizyInputFile         string
	laizyInputFileContents string
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
		"Type %load to load prompt from a file",
		"Type %save to save the current output to a file",
		"Type %exec to execute a shell command",
		"Type %history to show the command history",
		"Type %multi to toggle multiline mode",
		"Type %chain to toggle chaining (prompt-output-prompt) mode",
	}
	bannerQOTD = ""
	clear      map[string]func() //create a map for storing clear funcs

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
	unmodifiedPrompt := userPrompt
	userPrompt = strings.Split(userPrompt, " ")[0]

	if userPrompt == "%multi" {
		laizyInputMultiLine = !laizyInputMultiLine
		if laizyInputMultiLine {
			pterm.Info.Println("Multi line input enabled")
		} else {
			pterm.Info.Println("Single line input enabled")
		}

		return true
	}
	if userPrompt == "%chain" {
		laizyInputChain = !laizyInputChain
		if laizyInputChain {
			pterm.Info.Println("Chained input enabled")
		} else {
			pterm.Info.Println("Chained input disabled")
		}

		return true
	}
	if userPrompt == "%clear" {
		CallClear()
		return true
	}
	if userPrompt == "%exit" || userPrompt == "%quit" || userPrompt == "exit" || userPrompt == "quit" {
		os.Exit(0)
		return true
	}
	if userPrompt == "%help" || userPrompt == "help" {

		for _, entry := range helpMenuEntries {
			pterm.Info.Println(entry)
		}

		return true
	}
	if userPrompt == "%history" {
		pterm.Info.Println("Laizy CLI Prompt History")
		for index, prompt := range promptHistory {
			pterm.Info.Println(index, prompt)
		}
		return true
	}
	if userPrompt == "%ld" || userPrompt == "%loaddata" {
		if len(strings.Split(unmodifiedPrompt, " ")) > 1 {
			laizyInputFile = strings.Split(unmodifiedPrompt, " ")[1]
			laizyInputFile = strings.TrimSpace(laizyInputFile)
		} else {
			laizyInputFile, _ = pterm.DefaultInteractiveTextInput.Show("Enter a filename to load the data from")
		}

		laizyInputFileContents, err := os.ReadFile(laizyInputFile)
		if err != nil {
			pterm.Error.Println("Error loading file", err)
		}
		laizyLastResponse = string(laizyInputFileContents)
		lastPrompt = laizyLastResponse
		laizyFullResponse = ""
		pterm.Println("loaded data from file\n", lastPrompt)
		return true
	}
	if userPrompt == userPrompt == "%lp" || "%load" {
		if len(strings.Split(unmodifiedPrompt, " ")) > 1 {
			laizyInputFile = strings.Split(unmodifiedPrompt, " ")[1]
			laizyInputFile = strings.TrimSpace(laizyInputFile)
		} else {
			laizyInputFile, _ = pterm.DefaultInteractiveTextInput.Show("Enter a filename to load the prompt from")
		}

		laizyInputFileContents, err := os.ReadFile(laizyInputFile)
		if err != nil {
			pterm.Error.Println("Error loading file", err)
		}
		promptValue = string(laizyInputFileContents)
		lastPrompt = promptValue
		laizyFullResponse = ""
		pterm.Println("loaded prompt from file\n", promptValue)
		return true
	}
	if userPrompt == "%exec" || userPrompt == "%!" {
		if len(strings.Split(unmodifiedPrompt, " ")) > 1 {
			baseCommand := strings.Split(unmodifiedPrompt, " ")[1]
			shellCommandWithArgs := strings.Split(unmodifiedPrompt, " ")[2:]
			out, err := exec.Command(baseCommand, shellCommandWithArgs...).Output()
			if err != nil {
				pterm.Error.Println(err)
			}
			pterm.Println(string(out))
		} else {
			pterm.Error.Println("No shell command provided")
		}
		return true
	}
	if userPrompt == "%save" || userPrompt == "%s" {
		var laizyOutputFile string
		if len(strings.Split(unmodifiedPrompt, " ")) > 1 {
			laizyOutputFile = strings.Split(unmodifiedPrompt, " ")[1]
		} else {
			laizyOutputFile, _ = pterm.DefaultInteractiveTextInput.Show("Enter a filename to save the last response to")
		}
		pterm.Info.Println("Saving prompt output to file", laizyOutputFile)
		f, err := os.OpenFile(laizyOutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			pterm.Error.Println("Error creating file", err, laizyOutputFile)
		}
		defer f.Close()
		_, err = io.WriteString(f, laizyFullResponse)
		if err != nil {
			pterm.Error.Println("Error writing to file", laizyOutputFile)
		}
		return true
	}
	if userPrompt == "%qotd" {
		pterm.Info.Println(bannerQOTD)
		return true
	}
	if regexp.MustCompile(`^%`).MatchString(userPrompt) {
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
	pterm.DefaultCenter.Println(pterm.NewStyle(pterm.FgLightMagenta).Sprint("Try some of these prompts:"))
	pterm.DefaultCenter.Println(pterm.NewStyle(pterm.FgLightMagenta).Sprint("Generate some golang code for a web server"))
	pterm.DefaultCenter.Println(pterm.NewStyle(pterm.FgLightMagenta).Sprint("Generate a bash script to install a web server"))
	pterm.DefaultCenter.Println(pterm.NewStyle(pterm.FgLightMagenta).Sprint("Generate a bash script to install a web server that uses a database"))

	for {
		var laizySpinnerMessage = "thinking"
		var laizyResponseJSON map[string]interface{}
		inputPromptStyle := pterm.NewStyle(pterm.FgLightYellow, pterm.BgLightBlue)
		var userPromptValue string
		var chainIcon string
		if laizyInputChain {
			chainIcon = "⛓"
		} else {
			chainIcon = ""
		}
		laizyPrompt := fmt.Sprintf("%sLAIZY>",chainIcon)
		if laizyInputMultiLine {
			userPromptValue, _ = pterm.DefaultInteractiveTextInput.WithMultiLine().WithTextStyle(inputPromptStyle).Show(laizyPrompt)
		} else {
			userPromptValue, _ = pterm.DefaultInteractiveTextInput.WithTextStyle(inputPromptStyle).Show(laizyPrompt)
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

			} else {
				// for loading data from a file
				if laizyInputFileContents != "" {
					pterm.Info.Println("Loading data from file", laizyInputFile)

					promptValue = userPromptValue
					laizyInputFileContents = ""
					laizyInputFile = ""
				} else {
					laizyFullResponse = ""

				}
			}
		} else {
			promptValue = userPromptValue
	
			lastPrompt = userPromptValue
			laizyFullResponse = ""
			promptHistory = append(promptHistory, userPromptValue)
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
			previousPrompt := promptHistory[len(promptHistory) -1 ]
			promptValue = previousPrompt +"\n" + laizyLastResponse + "\n" + userPromptValue
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
