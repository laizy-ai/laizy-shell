package main

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/pterm/pterm"
)

// UI notes
// when prompt is blank but user had previous prompts continue prompt plus output on enter
// when prompt is blank and user has no previous prompts show help

var (
	laizyAPIKey       = os.Getenv("LAIZY_API_KEY")
	promptHistory     = []string{}
	promptValue       = ""
	lastPrompt        = ""
	laizyFullResponse = ""
	laizyLastResponse = ""
	laizyQOTDMessages = []string{
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
	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["darwin"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	rand.Seed(time.Now().Unix())
	randomBannerMessage := laizyQOTDMessages[rand.Intn(len(laizyQOTDMessages))]
	bannerQOTD = pterm.NewStyle(pterm.FgLightMagenta).Sprint(randomBannerMessage)
	if laizyAPIKey == "" {
		pterm.Error.Println("Laizy API Key not set")
	}

}
func specialCommandHandler(userPrompt string) bool {
	if userPrompt == "%clear" {
		CallClear()
		return true
	}
	if userPrompt == "%exit" || userPrompt == "%quit" || userPrompt == "exit" || userPrompt == "quit" {
		os.Exit(0)
		return true
	}
	if userPrompt == "%help" {
		pterm.Info.Println("Laizy CLI")
		pterm.Info.Println("Type 'exit' to exit")
		pterm.Info.Println("Type 'clear' to clear the screen")
		pterm.Info.Println("Type 'help' to show this help")
		return true
	}
	if userPrompt == "%save" {
		laizyOutputFile, _ := pterm.DefaultInteractiveTextInput.Show("Enter a filename to save the last response to: ")
		pterm.Info.Println("Saving prompt output to file", laizyOutputFile)
		f, err := os.Create(laizyOutputFile)
		if err != nil {
			pterm.Error.Println("Error creating file", laizyOutputFile)
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

	return false
}

func main() {
	// clear screen on linux
	CallClear()
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

	// main repl loop
	for {
		var laizyResponseJSON map[string]interface{}
		// show user prompt
		// text style for a blue background

		inputPromptStyle := pterm.NewStyle(pterm.FgLightMagenta, pterm.BgLightBlue)
		userPromptValue, _ := pterm.DefaultInteractiveTextInput.WithTextStyle(inputPromptStyle).Show("LAIZY>")
		if specialCommandHandler(userPromptValue) {
			continue
		}

		// if user prompt is blank, show help
		if len(userPromptValue) == 0 {
			if len(laizyLastResponse) != 0 {
				// continue from last prompt
				promptValue = lastPrompt + laizyFullResponse
				// clear screen

			} else {
				promptValue = userPromptValue
			}
		} else {
			// log.Println("Default execution path")
			promptValue = userPromptValue
			lastPrompt = userPromptValue
			laizyFullResponse = ""
			promptHistory = append(promptHistory, userPromptValue)
		}
		// show the loading spinner
		spinnerInfo, _ := pterm.DefaultSpinner.Start("Laizy is thinking...")
		// send the request to laizy.dev
		laizyResponse := sendLaizyRequest(promptValue, 1)
		// retrieve the response from laizy.dev
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

// A JSON blob containing all us presidents and their party affiliation
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
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		pterm.Error.Println(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		pterm.Error.Println(err)
	}
	return (string(body))
}
