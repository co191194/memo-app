package service

import (
	"github.com/co191194/memo-app/internal/memo"
)

type MemoLoader interface {
	LoadMemos(path string) ([]memo.Memo, error)
}

type MemoService struct {
	MemoPath string
	Loader   MemoLoader
}

func (s *MemoService) ListMemos() ([]memo.Memo, error) {
	return s.Loader.LoadMemos(s.MemoPath)
}

func (s *MemoService) GetMemo(id int) (memo.Memo, error) {

	memos, err := s.Loader.LoadMemos(s.MemoPath)
	if err != nil {
		return memo.Memo{}, err
	}

	return memo.FindById(memos, id)
}
