package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Author struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func HandleGitlabAPI(apiPath string, method string, payload *strings.Reader) ([]byte, error) {
	gitlabBaseURL := os.Getenv("GITLAB_BASE_URL")
	privateToken := os.Getenv("GITLAB_PRIVATE_TOKEN")

	url := gitlabBaseURL + apiPath

	var req *http.Request
	var err error
	if method == "POST" {
		req, err = http.NewRequest(method, url, payload)
		req.Header.Set("Content-Type", "application/json")
		fmt.Println(req)
	} else if apiPath == "/user" {
		req, err = http.NewRequest(method, gitlabBaseURL, nil)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	req.Header.Add("PRIVATE-TOKEN", privateToken)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func CheckAuthUser() (Author, error) {
	body, err := HandleGitlabAPI("/user", "GET", nil)
	if err != nil {
		return Author{}, err
	}

	var usr Author
	err = json.Unmarshal(body, &usr)
	if err != nil {
		return Author{}, err
	}

	return usr, nil
}

type mergeRequest struct {
	Title      string `json:"title"`
	Created_At string `json:"created_at"`
	Author     Author `json:"author"`
}

func FetchMergeRequests(apiPath string) ([]mergeRequest, error) {
	body, err := HandleGitlabAPI(apiPath, "GET", nil)
	if err != nil {
		return nil, err
	}

	var mrs []mergeRequest
	err = json.Unmarshal(body, &mrs)
	if err != nil {
		return nil, err
	}

	return mrs, nil
}

type MergePayload struct {
	Source_Branch string `json:"source_branch"`
	Target_Branch string `json:"target_branch"`
	Title         string `json:"title"`
	Assignee_ID   int32  `json:"assignee_id"`
	Description   string `json:"description"`
	Reviewer_IDs  []int  `json:"reviewer_ids"`
}

func CreateGitlabMergeRequest(apiPath string, content string, ticket string, title string, assigneeID int32, reviewerIDs []int) error {
	templatePath := "./migrations/template.md"
	sourceBranch := GetGitBranch()

	formattedMarkdown, err := replaceTemplateContent(templatePath, content, ticket)
	if err != nil {
		panic(err)
	}

	if err := generateMarkdownFile(formattedMarkdown); err != nil {
		panic(err)
	}

	payload := MergePayload{
		Source_Branch: strings.TrimSpace(string(sourceBranch)),
		Target_Branch: "stage",
		Title:         title,
		Assignee_ID:   assigneeID,
		Description:   formattedMarkdown,
		Reviewer_IDs:  reviewerIDs,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Failed to marshal payload: $v", err)
	}

	response, err := HandleGitlabAPI(apiPath, "POST", strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to send merge request to GitLab: %v", err)
	}

	fmt.Printf("Gitlab response: %s\n", response)

	return nil
}

func replaceTemplateContent(templatePath, descriptions, ticketInput string) (string, error) {
	contentBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	content := string(contentBytes)

	descriptionLines := strings.Split(descriptions, "\n")
	for i, line := range descriptionLines {
		descriptionLines[i] = "- " + line
	}
	formattedDescription := strings.Join(descriptionLines, "\n")

	content = strings.Replace(content, "${1}", formattedDescription, 1)
	content = strings.Replace(content, "${2}", ticketInput, 1)

	return content, nil
}

func generateMarkdownFile(dynamicContent string) error {
	branchName := GetGitBranch()

	safeFileName := strings.ReplaceAll(branchName, "/", "-")

	baseDir := "migrations"

	filePath := filepath.Join(baseDir, safeFileName+".md")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filePath, []byte(dynamicContent), 0644); err != nil {
		return err
	}

	return nil
}

func GetGitBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Sprintf("%s", err)
	}

	return strings.TrimSpace(out.String())
}

func TimeSince(created time.Time) string {
	now := time.Now()
	duration := now.Sub(created)
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	return fmt.Sprintf("%dd %dh ago", days, hours)
}

func TruncateString(str string, num int) string {
	if len(str) > num {
		if num > 3 {
			return str[:num-3] + "..."
		} else {
			return str[:num]
		}
	}
	return str
}
