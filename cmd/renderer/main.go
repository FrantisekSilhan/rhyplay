package main

import (
	"fmt"
	"os"
	"rhyplay/internal/config"
	"rhyplay/internal/parser"
	"rhyplay/internal/render"
	"rhyplay/internal/utils"
	"strings"
	"time"
)

const (
	AppName    = "rhyplay"
	Version    = "v0.1.0"
	Repository = "github.com/FrantisekSilhan/rhyplay"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

func main() {
	configPath := "settings.json"
	changed, err := config.Load(configPath)
	if err != nil {
		printHeader()
		fmt.Printf("\n %s[X] CONFIG ERROR%s\n", ColorRed, ColorReset)
		fmt.Printf("     The settings file '%s' is corrupted or invalid.\n", configPath)
		fmt.Printf("     Details: %v\n", err)
		fmt.Printf("\n     Please fix it manually or delete it to reset.\n")
		os.Exit(1)
	}

	if changed {
		printHeader()
		fmt.Printf("\n %s[!] CONFIG UPDATED%s\n", ColorYellow, ColorReset)
		fmt.Printf("     '%s' was missing or outdated.\n", configPath)
		fmt.Printf("     It has been updated with default values.\n")
		fmt.Printf("\n     Please check the file and restart the application.\n")
		os.Exit(0)
	}

	if len(os.Args) < 3 {
		printHeader()
		fmt.Println("\nUsage:")
		fmt.Println("  rhyplay <replay.rhr> <map_file>")
		fmt.Println("\nArguments:")
		fmt.Println("  <replay.rhr>   The replay file")
		fmt.Println("  <map_file>     The beatmap file (supports: .rhm, .sspm)")
		os.Exit(1)
	}

	replayPath := os.Args[1]
	mapPath := os.Args[2]

	printHeader()
	fmt.Print("\n [1/3] Parsing files... ")
	start := time.Now()

	replayData, err := parser.ParseReplay(replayPath)
	if err != nil {
		fmt.Printf("\nError parsing replay: %v\n", err)
		os.Exit(1)
	}

	mapData, audioBuffer, err := parser.ParseMap(mapPath)
	if err != nil {
		fmt.Printf("\nError parsing map: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Done (%v)\n", time.Since(start).Truncate(time.Millisecond))

	totalDuration := float64(replayData.Frames[len(replayData.Frames)-1].Progress) / 1000.0
	fmt.Printf("       > Map:    %s\n", mapData.Title)
	fmt.Printf("       > Length: %.2fs\n", totalDuration)
	fmt.Printf("       > Notes:  %d\n", len(mapData.Notes))

	fmt.Print(" [2/3] Extracting audio... ")
	start = time.Now()
	audioFile, err := utils.SaveTempFile(audioBuffer, "audio-*.mp3")
	if err != nil {
		fmt.Printf("\nError saving audio: %v\n", err)
		os.Exit(1)
	}
	defer audioFile.Cleanup()
	fmt.Printf("Done (%v)\n", time.Since(start).Truncate(time.Millisecond))

	fmt.Println(" [3/3] Rendering video... ")
	renderer := render.NewRenderer(mapData, replayData)

	start = time.Now()
	err = renderer.Render("output.mp4", audioFile.Path)
	if err != nil {
		fmt.Printf("\n\n [X] RENDER ERROR\n")
		fmt.Printf("     %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n\n")
	fmt.Println("-----------------------------------------------------------")
	fmt.Printf(" Success! Render finished in %s\n", time.Since(start).Truncate(time.Second))
	fmt.Println(" Saved to: output.mp4")
	fmt.Println("-----------------------------------------------------------")
}

func handleConfigError(path string, err error) {
	if strings.Contains(err.Error(), "corrupted") {
		fmt.Printf("ERROR: Settings file '%s' is corrupted.\n", path)
		fmt.Println("Details:", err)
	} else {
		fmt.Println("Unexpected error loading config:", err)
	}
	os.Exit(1)
}

func printHeader() {
	fmt.Println("-----------------------------------------------------------")
	fmt.Printf("--- %s%s%s %s | %s ---\n", ColorCyan, AppName, ColorReset, Version, Repository)
	fmt.Println("-----------------------------------------------------------")
}
