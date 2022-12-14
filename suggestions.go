package main

var promptSuggestions = []string{
	"LAIZY>: %ld <path to iam role>\n💾LAIZY>: generate terraform code for the role above",
	"LAIZY>: %ld <path to code snippet>\n💾LAIZY>: Refactor the code snippet above to <...>",
	"LAIZY>: Generate a bash script to install a web server",
	"💾LAIZY>: Convert to <snake_case|camelCase|PascalCase|kebab-case|dot.case|UPPERCASE|lowercase>",
	"⛓LAIZY>: Convert the data above to <JSON|YAML|TOML|XML...>",
	"⛓LAIZY>: %execs lynx -dump -nolist <url>\n⛓LAIZY>: Summarize the content above",
	"LAIZY>: Generate terraform code for <insert AWS service here>",
	"LAIZY>: Generate terraform code for <insert GCP service here>",
	"LAIZY>: Generate terraform code for <insert Azure service here>",
	"LAIZY>: Generate a skeleton for a golang application including package main, import and func main: ",
	"LAIZY>: Generate a skeleton for a javascript application using the ECMAScript 2020 standard: ",

	"💡 Use %chain mode to allow Laizy to treat the last response as data",
	"💡 Use the %save command to save the last response to a file",
	"💡 Use the %ld to load data from a file, and process it with Laizy",
	"💡 Use the %lp command to load a prompt from a file",
	"💡 Load data into Laizy to by using the %execs, %chain, or %ld commands",
	"💡 Laizy can be used to generate code for any language",
	"💡 Laizy can translate between written languages",
	"💡 When in doubt, use the %help command",
	"💡 Use %fetch to get data from the internet",
	"💡 For basic information lookup, laizy can replacesgoogle search",
	"💡 Use %chain mode after %fetch, %ld, or %execs to process the data",
	"💡 Combine %fetch and %chain to get data from the internet and process it",
	"💡 Combine %execs and %chain to get data from the shell and process it",
	"💡 if Laizy starts going of the rails use the %forget command to clear short term memory",
}
