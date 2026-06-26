package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"rhyplay/internal/config"
	"rhyplay/internal/db"
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
	configPath := "settings.json"
	var (
		replayPath string
		mapPath    string
		stars      float64
	)

	flag.StringVar(&replayPath, "r", "", "Path to the replay file")
	flag.StringVar(&replayPath, "replay", "", "Path to the replay file")

	flag.StringVar(&mapPath, "m", "", "Path to the map file")
	flag.StringVar(&mapPath, "map", "", "Path to the map file")

	flag.Float64Var(&stars, "s", 0.0, "Map star rating")
	flag.Float64Var(&stars, "stars", 0.0, "Map star rating")

	flag.Usage = func() {
		printHeader()
		fmt.Println("\nUsage: rhyplay [flags]")
		fmt.Println("\nFlags:")
		fmt.Println("  -r, --replay <file>    The replay file (.rhr)")
		fmt.Println("  -m, --map    <file>    The beatmap file (.rhm, .sspm)")
		fmt.Println("  -s, --stars  <value>   Override star rating")

		fmt.Printf("\n%sModes of Operation:%s\n", ColorCyan, ColorReset)
		fmt.Println("  1. Automatic:  Provide only -r. rhyplay will find the map and stars")
		fmt.Println("                 in the Rhythia database (requires GamePath in settings).")
		fmt.Println("  2. Manual:     Provide -r and -m. Stars will be looked up in the DB")
		fmt.Println("                 as a fallback if the map format doesn't provide them.")
		fmt.Println("  3. Portable:   Provide -r, -m, and -s. This mode is completely")
		fmt.Println("                 independent of the game database.")

		fmt.Printf("\n%sConfiguration:%s\n", ColorYellow, ColorReset)
		fmt.Printf("  Settings file: %s (Edit this to change the GamePath)\n", configPath)
	}

	flag.Parse()

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

	if replayPath == "" {
		printHeader()
		fmt.Printf("\n %s[X] MISSING ARGUMENT%s\n", ColorRed, ColorReset)
		fmt.Printf("     The %s-r/--replay%s flag is required to start rendering.\n", ColorCyan, ColorReset)
		fmt.Printf("\n     %sExample:%s\n", ColorYellow, ColorReset)
		fmt.Printf("     rhyplay -r MyReplay.rhr\n")
		fmt.Printf("\n     For all options, run: %s--help%s\n", ColorCyan, ColorReset)
		os.Exit(1)
	}

	printHeader()

	fmt.Print("\n [1/4] Discovering assets... ")
	start := time.Now()

	replayData, err := parser.ParseReplay(replayPath)
	if err != nil {
		fmt.Printf("\nError parsing replay: %v\n", err)
		os.Exit(1)
	}

	var mapData *parser.MapData
	var audioBuffer []byte
	var dbMatchedStars float64

	mapIsCachedJson := false
	var finalMapSourcePath string

	if mapPath != "" {
		finalMapSourcePath = mapPath
	} else {
		dbPath := filepath.Join(config.Current.GamePath, "rhythia.db")
		dbMap, err := db.FindMap(dbPath, replayData.ScoreData.MapID, replayData.ScoreData.LegacyMapID)
		if err != nil {
			fmt.Printf("\n%sCould not find map in DB. Please provide map manually with -m%s\n", ColorRed, ColorReset)
			os.Exit(1)
		}
		finalMapSourcePath = filepath.Join(config.Current.GamePath, dbMap.Path)
		audioBuffer, _ = os.ReadFile(filepath.Join(config.Current.GamePath, dbMap.AudioPath))
		dbMatchedStars = dbMap.StarRating
		mapIsCachedJson = true
	}
	fmt.Printf("Done (%v)\n", time.Since(start).Truncate(time.Millisecond))

	fmt.Print(" [2/4] Parsing map data... ")
	start = time.Now()

	if mapIsCachedJson {
		mapJson, err := os.ReadFile(finalMapSourcePath)
		if err != nil {
			fmt.Printf("\nError reading cached map: %v\n", err)
			os.Exit(1)
		}
		err = json.Unmarshal(mapJson, &mapData)
		if err != nil {
			fmt.Printf("\nError decoding cached map: %v\n", err)
			os.Exit(1)
		}
		mapData.Normalize()
		mapData.StarRating = dbMatchedStars
	} else {
		var mapAudio []byte
		mapData, mapAudio, err = parser.ParseMap(finalMapSourcePath)
		if err != nil {
			fmt.Printf("\nError parsing map file: %v\n", err)
			os.Exit(1)
		}
		audioBuffer = mapAudio

		if mapData.StarRating == 0 && stars == 0 {
			dbPath := filepath.Join(config.Current.GamePath, "rhythia.db")
			if m, err := db.FindMap(dbPath, replayData.ScoreData.MapID, replayData.ScoreData.LegacyMapID); err == nil {
				mapData.StarRating = m.StarRating
			}
		}
	}

	if stars > 0 {
		mapData.StarRating = stars
	}

	fmt.Printf("Done (%v)\n", time.Since(start).Truncate(time.Millisecond))

	totalDuration := float64(replayData.Frames[len(replayData.Frames)-1].Progress) / 1000.0
	fmt.Printf("       > Player: %s\n", replayData.ScoreData.PlayerName)
	fmt.Printf("       > Map:    %s\n", mapData.Title)
	fmt.Printf("       > Length: %.2fs\n", totalDuration)
	fmt.Printf("       > Notes:  %d\n", len(mapData.Notes))
	fmt.Printf("       > Stars:  %.2f\n", mapData.StarRating)

	fmt.Print(" [3/4] Preparing audio... ")
	if len(audioBuffer) == 0 {
		fmt.Printf("\n%sError: No audio found for this map.%s\n", ColorRed, ColorReset)
		os.Exit(1)
	}

	start = time.Now()
	audioFile, err := utils.SaveTempFile(audioBuffer, "audio-*.mp3")
	if err != nil {
		fmt.Printf("\nError saving audio: %v\n", err)
		os.Exit(1)
	}
	defer audioFile.Cleanup()
	fmt.Printf("Done (%v)\n", time.Since(start).Truncate(time.Millisecond))

	if _, err := os.Stat("output"); os.IsNotExist(err) {
		_ = os.Mkdir("output", 0755)
	}

	fmt.Println(" [4/4] Rendering video... ")
	renderer, err := render.NewRenderer(mapData, replayData)
	if err != nil {
		fmt.Printf("\nError initializing renderer: %v\n", err)
		os.Exit(1)
	}

	outputPath := fmt.Sprintf("output/%s_%s_%d.mp4",
		sanitizePart(replayData.ScoreData.PlayerName),
		sanitizePart(mapData.Title),
		replayData.ScoreData.Timestamp)

	start = time.Now()
	err = renderer.Render(outputPath, audioFile.Path)
	if err != nil {
		fmt.Printf("\n\n %s[X] RENDER ERROR%s\n", ColorRed, ColorReset)
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
