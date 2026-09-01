package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

var errProblemNotFound = errors.New("problem not found")

func htmlToText(htmlString string) string {
	doc, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		return htmlString
	}

	var builder strings.Builder
	atLineStart := true
	noSpaceBeforeNext := false

	appendText := func(text string) {
		words := strings.Fields(text)
		for _, word := range words {
			if !atLineStart && !noSpaceBeforeNext {
				builder.WriteString(" ")
			}
			builder.WriteString(word)
			atLineStart = false
			noSpaceBeforeNext = false
		}
	}

	appendNewline := func() {
		current := builder.String()
		if current == "" || strings.HasSuffix(current, "\n\n") {
			atLineStart = true
			return
		}
		if strings.HasSuffix(current, "\n") {
			builder.WriteString("\n")
		} else {
			builder.WriteString("\n\n")
		}
		atLineStart = true
	}

	appendLineBreak := func() {
		current := builder.String()
		if current == "" || strings.HasSuffix(current, "\n") {
			atLineStart = true
			return
		}
		builder.WriteString("\n")
		atLineStart = true
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}

		if n.Type == html.TextNode {
			appendText(n.Data)
			return
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "br":
				appendLineBreak()
				return
			case "p", "div", "pre", "ul", "ol":
				appendNewline()
				defer appendNewline()
			case "li":
				appendLineBreak()
				builder.WriteString("- ")
				atLineStart = false
				noSpaceBeforeNext = true
				defer appendLineBreak()
			case "sup":
				builder.WriteString("^")
				atLineStart = false
				noSpaceBeforeNext = true
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(doc)
	return strings.TrimSpace(builder.String())
}

func handleProblem(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		writeJSONError(w, http.StatusBadRequest, "problem slug is required")
		return
	}

	problem, err := leetcodeProblem(r.Context(), slug)
	if err != nil {
		if errors.Is(err, errProblemNotFound) {
			writeJSONError(w, http.StatusNotFound, "problem not found")
			return
		}
		fmt.Printf("problem error: %s\n", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to connect to leetcode. try again")
		return
	}
	writeJSON(w, http.StatusOK, problem)
}

type topicTag struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type problemResponse struct {
	QuestionFrontendID string     `json:"questionFrontendId"`
	Title              string     `json:"title"`
	TitleSlug          string     `json:"titleSlug"`
	Difficulty         string     `json:"difficulty"`
	IsPaidOnly         bool       `json:"isPaidOnly"`
	ACRate             float64    `json:"acRate"`
	Likes              int        `json:"likes"`
	Dislikes           int        `json:"dislikes"`
	Content            string     `json:"content"`
	TopicTags          []topicTag `json:"topicTags"`
}

type graphQLProblemResponse struct {
	Data struct {
		Question *problemResponse `json:"question"`
	} `json:"data"`

	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func leetcodeProblem(ctx context.Context, slug string) (problemResponse, error) {
	query := `query getProblem($titleSlug: String!) {
		question(titleSlug: $titleSlug) {
			questionFrontendId
			title
			titleSlug
			difficulty
			isPaidOnly
			acRate
			likes
			dislikes
			content
			topicTags {
				name
				slug
			}
		}
	}`

	body := graphQLRequest{
		Query: query,
		Variables: map[string]any{
			"titleSlug": slug,
		},
	}
	var result graphQLProblemResponse

	err := postGraphQL(ctx, body.Query, body.Variables, &result)
	if err != nil {
		return problemResponse{}, err
	}
	if result.Data.Question == nil {
		return problemResponse{}, errProblemNotFound
	}
	if len(result.Errors) > 0 {
		return problemResponse{}, fmt.Errorf("leetcode graphql error: %s", result.Errors[0].Message)
	}
	problem := *result.Data.Question
	problem.Content = htmlToText(problem.Content)

	return problem, nil
}
