package service_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/co191194/memo-app/internal/memo"
	"github.com/co191194/memo-app/internal/service"
	"github.com/co191194/memo-app/internal/storage"
)

type Memo = memo.Memo

var realMemoOperator = storage.StorageOperatorImpl{}

// saveTestMemos は一時ディレクトリにメモを保存し、そのファイルパスを返す。
func saveTestMemos(t *testing.T, memos []Memo) string {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), "memos.json")
	if err := realMemoOperator.SaveMemos(filePath, memos); err != nil {
		t.Fatalf("SaveMemos() err = %v", err)
	}
	return filePath
}

func assertEqualsMemo(t *testing.T, actual Memo, expected Memo) {
	t.Helper()

	if actual.ID != expected.ID {
		t.Errorf("ID = %d, expected = %d", actual.ID, expected.ID)
	}
	if actual.Title != expected.Title {
		t.Errorf("Title = %q, expected = %q", actual.Title, expected.Title)
	}
	if actual.Body != expected.Body {
		t.Errorf("Body = %q, expected = %q", actual.Body, expected.Body)
	}
	if !actual.CreatedAt.Equal(expected.CreatedAt) {
		t.Errorf("CreatedAt = %v, expected = %v", actual.CreatedAt, expected.CreatedAt)
	}
	if !actual.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, expected = %v", actual.UpdatedAt, expected.UpdatedAt)
	}
}

func TestMemoService_ListMemos(t *testing.T) {
	createdAt := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 10, 2, 18, 30, 0, 0, time.UTC)

	// IDの昇順ではなく、保存した順で返ることを確認するため、IDを入れ替えて保存する
	memos := []Memo{
		{ID: 3, Title: "Third", Body: "本文3", CreatedAt: createdAt, UpdatedAt: updatedAt},
		{ID: 1, Title: "First", Body: "本文1\n2行目", CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: 2, Title: "Second", Body: "", CreatedAt: updatedAt, UpdatedAt: updatedAt},
	}
	filePath := saveTestMemos(t, memos)

	svc := service.MemoService{
		MemoPath: filePath,
		Loader:   &realMemoOperator,
	}

	actual, err := svc.ListMemos()
	if err != nil {
		t.Fatalf("ListMemos() err = %v", err)
	}

	if len(actual) != len(memos) {
		t.Fatalf("len = %d, expected = %d", len(actual), len(memos))
	}
	for i := range memos {
		assertEqualsMemo(t, actual[i], memos[i])
	}
}

func TestMemoService_GetMemo(t *testing.T) {
	createdAt := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 10, 2, 18, 30, 0, 0, time.UTC)

	target := Memo{ID: 2, Title: "Target", Body: "対象の本文\n2行目", CreatedAt: createdAt, UpdatedAt: updatedAt}
	memos := []Memo{
		{ID: 1, Title: "Other", Body: "", CreatedAt: createdAt, UpdatedAt: createdAt},
		target,
		{ID: 3, Title: "Other", Body: "", CreatedAt: createdAt, UpdatedAt: createdAt},
	}
	filePath := saveTestMemos(t, memos)

	svc := service.MemoService{
		MemoPath: filePath,
		Loader:   &realMemoOperator,
	}

	actual, err := svc.GetMemo(2)
	if err != nil {
		t.Fatalf("GetMemo() err = %v", err)
	}

	assertEqualsMemo(t, actual, target)
}

// fakeFailLoadMemoOperator は読み込みに必ず失敗するLoader。
type fakeFailLoadMemoOperator struct {
	err error
}

func (f *fakeFailLoadMemoOperator) LoadMemos(path string) ([]Memo, error) {
	return nil, f.err
}

func TestMemoService_GetMemo_NotFound(t *testing.T) {
	createdAt := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

	t.Run("存在しないIDの場合", func(t *testing.T) {
		filePath := saveTestMemos(t, []Memo{
			{ID: 1, Title: "Other", Body: "", CreatedAt: createdAt, UpdatedAt: createdAt},
			{ID: 3, Title: "Other", Body: "", CreatedAt: createdAt, UpdatedAt: createdAt},
		})

		svc := service.MemoService{
			MemoPath: filePath,
			Loader:   &realMemoOperator,
		}

		_, err := svc.GetMemo(2)
		if !errors.Is(err, memo.ErrNotFound) {
			t.Errorf("GetMemo() err = %v, expected memo.ErrNotFound", err)
		}
	})

	t.Run("保存先ファイルがない場合", func(t *testing.T) {
		svc := service.MemoService{
			MemoPath: filepath.Join(t.TempDir(), "memos.json"),
			Loader:   &realMemoOperator,
		}

		_, err := svc.GetMemo(1)
		if !errors.Is(err, memo.ErrNotFound) {
			t.Errorf("GetMemo() err = %v, expected memo.ErrNotFound", err)
		}
	})
}

func TestMemoService_ListMemos_FileNotExist(t *testing.T) {
	svc := service.MemoService{
		MemoPath: filepath.Join(t.TempDir(), "memos.json"),
		Loader:   &realMemoOperator,
	}

	actual, err := svc.ListMemos()
	if err != nil {
		t.Fatalf("ListMemos() err = %v", err)
	}
	if len(actual) != 0 {
		t.Errorf("len = %d, expected = 0", len(actual))
	}
}

func TestMemoService_LoadFailure(t *testing.T) {
	loadErr := errors.New("load failed")

	svc := service.MemoService{
		MemoPath: "memos.json",
		Loader:   &fakeFailLoadMemoOperator{err: loadErr},
	}

	t.Run("ListMemos", func(t *testing.T) {
		_, err := svc.ListMemos()
		assertLoadFailure(t, err, loadErr)
	})

	t.Run("GetMemo", func(t *testing.T) {
		_, err := svc.GetMemo(1)
		assertLoadFailure(t, err, loadErr)
	})
}

// assertLoadFailure は、読み込み失敗のエラーが元のエラーを保ったまま返り、
// 「メモが存在しない」とは区別できることを確認する。
func assertLoadFailure(t *testing.T, actual error, loadErr error) {
	t.Helper()

	if !errors.Is(actual, loadErr) {
		t.Errorf("err = %v, expected to wrap %v", actual, loadErr)
	}
	if errors.Is(actual, memo.ErrNotFound) {
		t.Errorf("err = %v, expected not to be memo.ErrNotFound", actual)
	}
}
