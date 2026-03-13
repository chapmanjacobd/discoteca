package main

import (
	"fmt"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// Example: Hybrid FTS Search Query Parsing
func main() {
	examples := []string{
		`python tutorial`,
		`"video tutorial"`,
		`python "video tutorial" beginner`,
		`"machine learning" "deep learning"`,
		`python OR golang "machine learning"`,
		`"ab" video`,  // "ab" ignored (< 3 chars)
		`'single quotes' work`,
	}

	fmt.Println("Hybrid FTS Search Query Parsing Examples")
	fmt.Println("=========================================\n")

	for _, query := range examples {
		fmt.Printf("Query: %s\n", query)
		
		hybrid := utils.ParseHybridSearchQuery(query)
		
		fmt.Printf("  FTS Terms: %v\n", hybrid.FTSTerms)
		fmt.Printf("  Phrases:   %v\n", hybrid.Phrases)
		
		if hybrid.HasFTSTerms() {
			ftsQuery := hybrid.BuildFTSQuery(" OR ")
			fmt.Printf("  FTS SQL:   media_fts MATCH '%s'\n", ftsQuery)
		}
		
		if hybrid.HasPhrases() {
			for i, phrase := range hybrid.Phrases {
				fmt.Printf("  LIKE[%d]:  (path LIKE '%%%s%%' OR title LIKE '%%%s%%' OR description LIKE '%%%s%%')\n",
					i, phrase, phrase, phrase)
			}
		}
		
		fmt.Println()
	}
}
