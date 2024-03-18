package gitlab

import (
	"errors"
	"fmt"
	"time"

	"github.com/xanzy/go-gitlab"
)

const RefreshInterval = time.Minute

type Result struct {
	AssignedIssues uint
	AssignedMRs    uint
	ReviewMRs      uint
	ToDos          uint
}

type Settings struct {
	PersonalAccessToken string `json:"personalAccessToken"`
	Server              string `json:"server"`
	Username            string `json:"-"`
	UserID              int    `json:"-"`
}

const (
	perPage        = 20
	paginationType = "keyset"
)

func FetchUnseenCount(settings *Settings) (Result, error) {
	if settings.PersonalAccessToken == "" {
		return Result{}, errors.New("missing PersonalAccessToken")
	}
	if settings.Server == "" {
		return Result{}, errors.New("missing Server")
	}

	return getUnreadCounts(settings)
}

func getUnreadCounts(settings *Settings) (Result, error) {
	git, err := gitlab.NewClient(settings.PersonalAccessToken, gitlab.WithBaseURL(settings.Server))
	if err != nil {
		return Result{}, fmt.Errorf("error while getting session: %w", err)
	}

	if settings.UserID == 0 || settings.Username == "" {
		user, _, err := git.Users.CurrentUser()
		if err != nil {
			return Result{}, fmt.Errorf("error while getting current user: %w", err)
		}
		settings.Username = user.Username
		settings.UserID = user.ID
	}

	result := Result{}

	assignedIssues, err := getAssignedIssues(git, settings.Username)
	if err != nil {
		return Result{}, err
	}
	result.AssignedIssues = assignedIssues

	assignedMRs, err := getAssignedMRs(git, gitlab.AssigneeID(settings.UserID))
	if err != nil {
		return Result{}, err
	}
	result.AssignedMRs = assignedMRs

	reviewMRs, err := getReviewMRs(git, gitlab.ReviewerID(settings.UserID))
	if err != nil {
		return Result{}, err
	}
	result.ReviewMRs = reviewMRs

	todos, err := getTodos(git)
	if err != nil {
		return Result{}, err
	}
	result.ToDos = todos

	return result, nil
}

func getAssignedIssues(git *gitlab.Client, username string) (uint, error) {
	var result uint

	options := gitlab.ListIssuesOptions{
		AssigneeUsername: gitlab.Ptr(username),
		State:            gitlab.Ptr("opened"),
		Scope:            gitlab.Ptr("all"),
		ListOptions: gitlab.ListOptions{
			PerPage:    perPage,
			Pagination: paginationType,
		},
	}
	var requestOptions []gitlab.RequestOptionFunc
	for {
		issues, response, err := git.Issues.ListIssues(&options, requestOptions...)
		if err != nil {
			return 0, fmt.Errorf("error while getting assigned issues: %w", err)
		}
		result += uint(len(issues))

		if response.NextLink == "" {
			break
		}

		requestOptions = []gitlab.RequestOptionFunc{
			gitlab.WithKeysetPaginationParameters(response.NextLink),
		}
	}

	return result, nil
}

func getAssignedMRs(git *gitlab.Client, userID *gitlab.AssigneeIDValue) (uint, error) {
	var result uint

	options := gitlab.ListMergeRequestsOptions{
		AssigneeID: userID,
		State:      gitlab.Ptr("opened"),
		Scope:      gitlab.Ptr("all"),
		ListOptions: gitlab.ListOptions{
			PerPage:    perPage,
			Pagination: paginationType,
		},
	}
	var requestOptions []gitlab.RequestOptionFunc
	for {
		issues, response, err := git.MergeRequests.ListMergeRequests(&options, requestOptions...)
		if err != nil {
			return 0, fmt.Errorf("error while getting assigned MRs: %w", err)
		}
		result += uint(len(issues))

		if response.NextLink == "" {
			break
		}

		requestOptions = []gitlab.RequestOptionFunc{
			gitlab.WithKeysetPaginationParameters(response.NextLink),
		}
	}

	return result, nil
}

func getReviewMRs(git *gitlab.Client, userID *gitlab.ReviewerIDValue) (uint, error) {
	var result uint

	options := gitlab.ListMergeRequestsOptions{
		ReviewerID: userID,
		State:      gitlab.Ptr("opened"),
		Scope:      gitlab.Ptr("all"),
		ListOptions: gitlab.ListOptions{
			PerPage:    perPage,
			Pagination: paginationType,
		},
	}
	var requestOptions []gitlab.RequestOptionFunc
	for {
		issues, response, err := git.MergeRequests.ListMergeRequests(&options, requestOptions...)
		if err != nil {
			return 0, fmt.Errorf("error while getting review MRs: %w", err)
		}
		result += uint(len(issues))

		if response.NextLink == "" {
			break
		}

		requestOptions = []gitlab.RequestOptionFunc{
			gitlab.WithKeysetPaginationParameters(response.NextLink),
		}
	}

	return result, nil
}

func getTodos(git *gitlab.Client) (uint, error) {
	var result uint

	options := gitlab.ListTodosOptions{
		ListOptions: gitlab.ListOptions{
			PerPage:    perPage,
			Pagination: paginationType,
		},
	}
	var requestOptions []gitlab.RequestOptionFunc
	for {
		issues, response, err := git.Todos.ListTodos(&options, requestOptions...)
		if err != nil {
			return 0, fmt.Errorf("error while getting todos: %w", err)
		}
		result += uint(len(issues))

		if response.NextLink == "" {
			break
		}

		requestOptions = []gitlab.RequestOptionFunc{
			gitlab.WithKeysetPaginationParameters(response.NextLink),
		}
	}

	return result, nil
}
