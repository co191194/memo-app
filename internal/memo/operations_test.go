package memo_test

import (
	"testing"
	"time"

	"github.com/co191194/memo-app/internal/memo"
)

type Memo = memo.Memo

func TestCreateMemo(t *testing.T) {
	now := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)

	memos := []Memo{
		{ID: 4, Title: "Exist1"},
		{ID: 1, Title: "Exist2"},
	}

	actual := memo.CreateMemo(memos, "New Memo", "New Body", now)

	if actual.ID != 5 {
		t.Errorf("ID = %d, expected = 5", actual.ID)
	}

	if actual.Title != "New Memo" {
		t.Errorf("Title = %q, expected = %q", actual.Title, "New Memo")
	}

	if actual.Body != "New Body" {
		t.Errorf("Body = %q, expected = %q", actual.Body, "New Body")
	}

	if !actual.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, expected = %v", actual.CreatedAt, now)
	}

	if !actual.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, expected = %v", actual.UpdatedAt, now)
	}
}

func TestFindById(t *testing.T) {
	memos := []Memo{
		{ID: 1, Title: "Other"},
		{ID: 3, Title: "Other"},
		{ID: 2, Title: "Target"},
		{ID: 4, Title: "Other"},
	}

	actual, err := memo.FindById(memos, 2)
	if err != nil {
		t.Fatalf("findById() err = %v", err)
	}

	if actual.ID != 2 {
		t.Errorf("ID = %d, expected = 2", actual.ID)
	}

	if actual.Title != "Target" {
		t.Errorf("Title = %q, expected = %q", actual.Title, "Target")
	}
}

func TestFindById_NotFound(t *testing.T) {
	memos := []Memo{
		{ID: 1, Title: "Other"},
		{ID: 3, Title: "Other"},
	}

	_, actual := memo.FindById(memos, 2)
	if actual == nil {
		t.Fatalf("findById any match")
	}

	assertEqualsMessage(t, actual.Error(), "memo not found: 2")
}

func TestFilterMemos(t *testing.T) {
	memos := []Memo{
		{ID: 1, Title: "Other Title1", Body: "Other Body1"},
		{ID: 2, Title: "Target Title2", Body: "Body2"},
		{ID: 3, Title: "Title3", Body: "Body3 Target"},
		{ID: 4, Title: "Other Title4", Body: "Other Body4"},
		{ID: 5, Title: "Title5 Target Middle", Body: "Body5"},
		{ID: 6, Title: "Other Title6", Body: "Other Body6"},
	}

	actual, err := memo.FilterMemos(memos, "Target")
	if err != nil {
		t.Fatalf("filterMemos() err = %v", err)
	}

	if len(actual) != 3 {
		t.Fatalf("actual len = %d, expected = 3", len(actual))
	}
	if actual[0].ID != 2 {
		t.Errorf("actual[0].ID = %d, expected = 2", actual[0].ID)
	}
	if actual[1].ID != 3 {
		t.Errorf("actual[1].ID = %d, expected = 3", actual[1].ID)
	}
	if actual[2].ID != 5 {
		t.Errorf("actual[2].ID = %d, expected = 5", actual[2].ID)
	}
}

func TestFilterMemos_NotMatch(t *testing.T) {
	memos := []Memo{
		{ID: 1, Title: "Other Title1", Body: "Other Body1"},
		{ID: 6, Title: "Other Title6", Body: "Other Body6"},
	}

	_, actual := memo.FilterMemos(memos, "Target")
	if actual == nil {
		t.Fatalf("filterMemos any match")
	}

	assertEqualsMessage(t, actual.Error(), "No matching memos found.")
}

func TestBuildDeletedMemos(t *testing.T) {
	memos := []Memo{
		{ID: 6, Title: "Other1"},
		{ID: 3, Title: "Target"},
		{ID: 1, Title: "Other2"},
	}

	actual, err := memo.BuildDeletedMemos(memos, 3)
	if err != nil {
		t.Fatalf("buildDeletedMemos() err = %v", err)
	}

	if len(actual) != 2 {
		t.Fatalf("actual.len = %d, expected = 2", len(actual))
	}

	if actual[0].ID != 6 {
		t.Errorf("actual[0].ID = %d, expected = 6", actual[0].ID)
	}

	if actual[1].ID != 1 {
		t.Errorf("actual[1].ID = %d, expected = 1", actual[1].ID)
	}
}

func TestBuildDeletedMemos_NotFountMemo(t *testing.T) {
	memos := []Memo{
		{ID: 6, Title: "Other1"},
		{ID: 1, Title: "Other2"},
	}

	_, actual := memo.BuildDeletedMemos(memos, 3)
	if actual == nil {
		t.Fatalf("buildDeletedMemos() any deleted memo")
	}

	assertEqualsMessage(t, actual.Error(), "memo not found: 3")
}

func assertEqualsMessage(t *testing.T, actual string, expected string) {
	t.Helper()
	if actual != expected {
		t.Errorf("actual = %q, expected = %q", actual, expected)
	}
}

func TestEditMemo(t *testing.T) {

	t.Run("異常系", func(t *testing.T) {
		t.Run("対象が見つからない場合", func(t *testing.T) {
			memos := []Memo{
				{ID: 1},
				{ID: 2},
				{ID: 4},
			}

			input := memo.EditMemoInput{
				ID: 3, Title: ptr("Edited Title"), Now: time.Date(2026, 9, 5, 13, 45, 0, 0, time.Local),
			}
			_, err := memo.EditMemo(memos, input)

			if err == nil {
				t.Fatal("EditMemo is not Error!")
			}

			assertEqualsMessage(t, err.Error(), "memo not found: 3")
		})

		t.Run("メモが保存されていない場合", func(t *testing.T) {
			memos := []Memo{}

			input := memo.EditMemoInput{
				ID: 1, Title: ptr("Edited Title"), Now: time.Date(2026, 9, 5, 13, 45, 0, 0, time.Local),
			}
			_, err := memo.EditMemo(memos, input)

			if err == nil {
				t.Fatal("EditMemo is not Error!")
			}

			assertEqualsMessage(t, err.Error(), "memo not found: 1")
		})
	})

	t.Run("正常系", func(t *testing.T) {
		testCases := []struct {
			testName      string
			title         *string
			body          *string
			expectedTitle string
			expectedBody  string
		}{
			{
				testName:      "Titleのみ更新",
				title:         ptr("Edited Title"),
				body:          nil,
				expectedTitle: "Edited Title",
				expectedBody:  "Target Body",
			},
			{
				testName:      "Bodyのみ更新",
				title:         nil,
				body:          ptr("Edited Body"),
				expectedTitle: "Target Title",
				expectedBody:  "Edited Body",
			},
			{
				testName:      "TitleとBodyの両方を更新",
				title:         ptr("Edited Title"),
				body:          ptr("Edited Body"),
				expectedTitle: "Edited Title",
				expectedBody:  "Edited Body",
			},
			{
				testName:      "Bodyを空文字で更新",
				title:         nil,
				body:          ptr(""),
				expectedTitle: "Target Title",
				expectedBody:  "",
			},
			{
				testName:      "同じ値で更新",
				title:         ptr("Target Title"),
				body:          ptr("Target Body"),
				expectedTitle: "Target Title",
				expectedBody:  "Target Body",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				createdAt := time.Date(2026, 9, 1, 10, 15, 0, 0, time.Local)
				memos := []Memo{
					{
						ID:        1,
						Title:     "Non-Target Title",
						Body:      "Non-Target Body",
						CreatedAt: createdAt,
						UpdatedAt: createdAt,
					},
					{
						ID:        2,
						Title:     "Target Title",
						Body:      "Target Body",
						CreatedAt: createdAt,
						UpdatedAt: createdAt,
					},
				}
				before := make([]Memo, len(memos))
				copy(before, memos)

				input := memo.EditMemoInput{
					ID:    2,
					Title: tc.title,
					Body:  tc.body,
					Now:   time.Date(2026, 9, 6, 15, 30, 0, 0, time.Local),
				}

				after, err := memo.EditMemo(memos, input)
				if err != nil {
					t.Fatalf("EditMemo is Error: %v", err)
				}

				if len(after) != 2 {
					t.Fatal("edited memos is not 2 elements")
				}

				for index := range memos {
					if memos[index] != before[index] {
						t.Errorf(
							"actual memos[%d] = %+v, expected memos[%d] = %+v",
							index,
							memos[index],
							index,
							before[index],
						)
					}
				}

				if after[0] != before[0] {
					t.Errorf(
						"actual after[0] = %+v, expected after[0] = %+v",
						after[0],
						before[0],
					)
				}

				if after[1].ID != 2 {
					t.Errorf(
						"actual ID = %d, expected ID = %d",
						after[1].ID,
						2,
					)
				}

				if after[1].Title != tc.expectedTitle {
					t.Errorf(
						"actual Title = %q, expected Title = %q",
						after[1].Title,
						tc.expectedTitle,
					)
				}

				if after[1].Body != tc.expectedBody {
					t.Errorf(
						"actual Body = %q, expected Body = %q",
						after[1].Body,
						tc.expectedBody,
					)
				}

				if !after[1].CreatedAt.Equal(createdAt) {
					t.Errorf(
						"actual CreatedAt = %q, expected CreatedAt = %q",
						after[1].CreatedAt.String(),
						createdAt.String(),
					)
				}

				if !after[1].UpdatedAt.Equal(time.Date(2026, 9, 6, 15, 30, 0, 0, time.Local)) {
					t.Errorf(
						"actual UpdatedAt = %q, expected UpdatedAt = %q",
						after[1].UpdatedAt.String(),
						time.Date(2026, 9, 6, 15, 30, 0, 0, time.Local).String(),
					)
				}
			})
		}
	})

}

func ptr[T any](v T) *T {
	return &v
}
