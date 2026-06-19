package main

import (
	"fmt"
	"os"
	"rhyplay/internal/config"
	"rhyplay/internal/parser"
	"rhyplay/internal/render"
	"rhyplay/internal/utils"
	"strings"
)

func main() {
	configPath := "settings.json"
	changed, err := config.Load(configPath)
	if err != nil {
		if strings.Contains(err.Error(), "corrupted") {
			fmt.Printf("ERROR: the settings file '%s' is corrupted or has invalid formatting\n", configPath)
			fmt.Println("details:", err)
			fmt.Println("\nplease fix the file manually or delete it to reset to defaults")
		} else {
			fmt.Println("unexpected error loading config:", err)
		}
		os.Exit(1)
	}

	if changed {
		fmt.Printf("NOTICE: '%s' was missing or outdated\n", configPath)
		fmt.Println("the file has been updated with default values for missing settings")
		fmt.Println("please check the settings and restart the application")
		os.Exit(0)
	}

	if len(os.Args) < 3 {
		fmt.Println("Usage: rhyplay <path_to_rhr_file> <path_to_rhm_file>")
		os.Exit(1)
	}

	replayPath := os.Args[1]
	fmt.Printf("Parsing replay: %s\n", replayPath)

	replayData, err := parser.ParseReplay(replayPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully parsed %d frames!\n", len(replayData.Frames))
	fmt.Println("----------------------------------------------------------------")
	fmt.Printf("%-8s | %-8s | %-8s | %-8s | %-5s\n", "Counter", "X", "Y", "Val", "Hit")
	fmt.Println("----------------------------------------------------------------")

	for i, f := range replayData.Frames {
		if i >= 20 {
			fmt.Println("...")
			break
		}
		fmt.Printf("%-8d | %-8.3f | %-8.3f | %-8.3f | %-5t\n",
			f.Counter, f.X, f.Y, f.Val, f.Hit)
	}

	// parse the map file
	mapPath := os.Args[2]
	fmt.Printf("\nParsing map: %s\n", mapPath)

	mapData, audioBuffer, err := parser.ParseMap(mapPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully parsed map: %s\n", mapData.Title)
	fmt.Printf("Notes:\n")
	for i, note := range mapData.Notes {
		if i >= 20 {
			fmt.Println("...")
			break
		}
		fmt.Printf("Time: %d, X: %.3f, Y: %.3f\n", note.Time, note.X, note.Y)
	}

	// extract audio
	audioFile, err := utils.SaveTempFile(audioBuffer, "audio-*.mp3")
	if err != nil {
		fmt.Printf("Error saving audio: %v\n", err)
		os.Exit(1)
	}
	defer audioFile.Cleanup()

	renderer := render.NewRenderer(mapData, replayData)
	err = renderer.Render("output.mp4", audioFile.Path)
	if err != nil {
		fmt.Printf("Error during rendering: %v\n", err)
		os.Exit(1)
	}
}
