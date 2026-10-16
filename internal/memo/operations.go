package memo

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("memo not found")

func newNotFoundError(id int) error {
	return fmt.Errorf("%w: %d", ErrNotFound, id)
}

func CreateMemo(memos []Memo, title string, body string, now time.Time) Memo {
	var id int
	if len(memos) > 0 {
		maxId := 0
		// JSONは順序を保証しないので全体を走査して最大IDを取得する
		for _, memo := range memos {
			if maxId < memo.ID {
				maxId = memo.ID
			}
		}
		id = maxId + 1
	} else {
		id = 1
	}

	return Memo{
		ID:        id,
		Title:     title,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func FindById(memos []Memo, id int) (Memo, error) {

	for _, memo := range memos {
		if memo.ID == id {
			return memo, nil
		}
	}

	return Memo{}, newNotFoundError(id)
}

func FilterMemos(memos []Memo, keyword string) ([]Memo, error) {
	filteredMemos := []Memo{}

	for _, memo := range memos {
		if strings.Contains(memo.Title, keyword) || strings.Contains(memo.Body, keyword) {
			filteredMemos = append(filteredMemos, memo)
		}
	}

	if len(filteredMemos) == 0 {
		return nil, errors.New("No matching memos found.")
	}

	return filteredMemos, nil
}

func BuildDeletedMemos(memos []Memo, id int) ([]Memo, error) {

	deletedMemos := []Memo{}

	for _, memo := range memos {
		if memo.ID != id {
			deletedMemos = append(deletedMemos, memo)
		}
	}

	if len(deletedMemos) == len(memos) {
		return nil, newNotFoundError(id)
	}

	return deletedMemos, nil
}

type EditMemoInput struct {
	ID    int
	Title *string
	Body  *string
	Now   time.Time
}

func EditMemo(memos []Memo, input EditMemoInput) ([]Memo, error) {

	editedMemos := make([]Memo, len(memos))
	copy(editedMemos, memos)

	// inputについて、正のID・少なくとも一方の指定・指定タイトルの妥当性は呼び出し側で検証済み
	index, err := findIndexById(editedMemos, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		editedMemos[index].Title = *input.Title
	}

	if input.Body != nil {
		editedMemos[index].Body = *input.Body
	}

	editedMemos[index].UpdatedAt = input.Now

	return editedMemos, nil
}

func findIndexById(memos []Memo, id int) (int, error) {
	for index, memo := range memos {
		if memo.ID == id {
			return index, nil
		}
	}

	return -1, newNotFoundError(id)
}
