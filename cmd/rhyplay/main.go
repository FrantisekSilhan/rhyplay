package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"rhyplay/internal/config"
	"rhyplay/internal/parser"
	"rhyplay/internal/render"
	"rhyplay/internal/utils"
	"strings"
	"time"
)

const (
	AppName    = "rhyplay"
	Repository = "github.com/FrantisekSilhan/rhyplay"
)

var (
	Version = "v0.0.0-dev"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

var illegalFileNameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func sanitizePart(s string) string {
	s = illegalFileNameChars.ReplaceAllString(s, "_")
	s = regexp.MustCompile(`_+`).ReplaceAllString(s, "_")
	return strings.Trim(s, " _")
}

func main() {
	starsPtr := flag.Float64("stars", 0.0, "Map star rating")
	flag.Usage = func() {
		printHeader()
		fmt.Println("\nUsage:")
		fmt.Println("  rhyplay [flags] <replay.rhr> <map_file>")
		fmt.Println("\nArguments:")
		fmt.Println("  <replay.rhr>   The replay file")
		fmt.Println("  <map_file>     The beatmap file (supports: .rhm, .sspm)")
		fmt.Println("\nOptional flags:")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()

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

	if len(args) > 2 {
		fmt.Println("[!] Found unexpected arguments. Ensure flags are placed before positional arguments.")
	}

	if len(args) < 2 {
		printHeader()
		fmt.Println("\nUsage:")
		fmt.Println("  rhyplay [flags] <replay.rhr> <map_file>")
		fmt.Println("\nArguments:")
		fmt.Println("  <replay.rhr>   The replay file")
		fmt.Println("  <map_file>     The beatmap file (supports: .rhm, .sspm)")
		fmt.Println("\nOptional flags:")
		fmt.Println("  --stars <value>  Override the map's star rating")
		os.Exit(1)
	}

	replayPath := args[0]
	mapPath := args[1]

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

	stars := *starsPtr
	if stars > 0.0 {
		mapData.StarRating = stars
	}

	totalDuration := float64(replayData.Frames[len(replayData.Frames)-1].Progress) / 1000.0
	fmt.Printf("       > Map:    %s\n", mapData.Title)
	fmt.Printf("       > Length: %.2fs\n", totalDuration)
	fmt.Printf("       > Notes:  %d\n", len(mapData.Notes))
	fmt.Printf("       > Stars:  %.2f\n", mapData.StarRating)

	fmt.Print(" [2/3] Extracting audio... ")
	start = time.Now()
	audioFile, err := utils.SaveTempFile(audioBuffer, "audio-*.mp3")
	if err != nil {
		fmt.Printf("\nError saving audio: %v\n", err)
		os.Exit(1)
	}
	defer audioFile.Cleanup()
	fmt.Printf("Done (%v)\n", time.Since(start).Truncate(time.Millisecond))

	if _, err := os.Stat("output"); os.IsNotExist(err) {
		err := os.Mkdir("output", 0755)
		if err != nil {
			fmt.Printf("\nError creating output directory: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println(" [3/3] Rendering video... ")
	renderer, err := render.NewRenderer(mapData, replayData)
	if err != nil {
		fmt.Printf("\nError initializing renderer: %v\n", err)
		os.Exit(1)
	}

	outputPath := fmt.Sprintf("output/%s_%s_%d.mp4", sanitizePart(replayData.ScoreData.PlayerName), sanitizePart(mapData.Title), replayData.ScoreData.Timestamp)

	start = time.Now()
	err = renderer.Render(outputPath, audioFile.Path)
	if err != nil {
		fmt.Printf("\n\n [X] RENDER ERROR\n")
		fmt.Printf("     %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n\n")
	separatorWidth := getHeaderWidth()
	printSeparator(separatorWidth)
	fmt.Printf(" Success! Render finished in %s\n", time.Since(start).Truncate(time.Second))
	fmt.Printf(" Saved to: %s\n", outputPath)
	printSeparator(separatorWidth)
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

func getHeaderWidth() int {
	return len(fmt.Sprintf("--- %s %s | %s ---", AppName, Version, Repository))
}

func printSeparator(width int) {
	fmt.Println(strings.Repeat("-", width))
}

func printHeader() {
	width := getHeaderWidth()
	printSeparator(width)
	fmt.Printf("--- %s%s%s %s | %s ---\n", ColorCyan, AppName, ColorReset, Version, Repository)
	printSeparator(width)
}
