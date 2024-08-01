# BubbleTea - README

## Overview

**BubbleTea** is an interactive command-line application built using the [Bubble Tea framework](https://github.com/charmbracelet/bubbletea) to streamline the creation of merge requests on GitLab. This application offers a user-friendly interface for filling in details such as the title, description, and reviewers of a merge request. It includes various input modes and a table display for managing existing merge requests.

## Installation

### Prerequisites
- Go 1.18+
- Git
- GitLab account with API access
- `.env` file for environment variables

### Running
1. git clone https://github.com/Saveljeffjonatan/os-turtle
2. cd os-turtle
3. go mod tidy
4. go run main.go

### Features
- Authentication Check: Verifies user credentials via CheckAuthUser.
- Interactive Merge Request Creation: Supports input for title, description, ticket ID, and reviewers.
- Merge Request Summary: Displays a summary of the merge request before submission.
- Error Handling: Provides feedback in case of issues.
- Vim like keybinds for navigation.

### Customization
- Edit migrations/template.md to your liking (this will be the content within your gitlab MR) *Remember that {1} and {2} is necessary for the app to work.
- Update your reviewers in bubbletea/bubble.go - 135.
