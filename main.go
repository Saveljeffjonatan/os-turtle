package main

import (
	"fmt"
	"log"
	"os"
	"time"
	bubble "turtle/bubbletea"
	t "turtle/utils"

	"github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
		return
	}

	_, err = t.CheckAuthUser()
	if err != nil {
		fmt.Printf("Error fetching auth user: %v\n", err)
		return
	}

	baseUrl := os.Getenv("GITLAB_PROJECT")

	mergeRequests := baseUrl + "/merge_requests?state=opened"
	mrs, err := t.FetchMergeRequests(mergeRequests)
	if err != nil {
		fmt.Printf("Error fetching merge requests: %v\n", err)
		return
	}

	bubbleteaMRs := make([]bubble.MergeRequest, len(mrs))
	for i, umr := range mrs {
		createdAt, err := time.Parse(time.RFC3339, umr.Created_At)
		if err != nil {
			log.Fatalf("Error parsing date: %v", err)
		}

		bubbleteaMRs[i] = bubble.MergeRequest{
			Title:     umr.Title,
			Author:    umr.Author.Name,
			CreatedAt: createdAt,
		}
	}

	tableModel := bubble.NewModel(bubbleteaMRs)
	p := tea.NewProgram(tableModel)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running table display: %v", err)
	}
}
