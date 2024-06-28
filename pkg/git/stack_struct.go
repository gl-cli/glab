package git

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type StackRef struct {
	Prev        string `json:"prev"`
	Branch      string `json:"branch"`
	SHA         string `json:"sha"`
	Next        string `json:"next"`
	MR          string `json:"mr"`
	Description string `json:"description"`
}

type Stack struct {
	Title string
	Refs  map[string]StackRef
}

func (s Stack) Empty() bool { return len(s.Refs) == 0 }

func (s *Stack) RemoveRef(ref StackRef) error {
	if ref.Next == "" && ref.Prev == "" {
		// this is the only ref, so just remove it
		err := DeleteStackRefFile(s.Title, ref)
		delete(s.Refs, ref.SHA)
		if err != nil {
			return fmt.Errorf("could not delete reference file %v:", err)
		}

		return nil
	}

	err := s.adjustAdjacentRefs(ref)
	if err != nil {
		return fmt.Errorf("error adjusting next reference %v:", err)
	}

	err = DeleteStackRefFile(s.Title, ref)
	if err != nil {
		return fmt.Errorf("could not delete reference file %v:", err)
	}

	err = s.RemoveBranch(ref)
	if err != nil {
		return fmt.Errorf("could not remove branch %v:", err)
	}

	delete(s.Refs, ref.SHA)

	return nil
}

func (s *Stack) RemoveBranch(ref StackRef) error {
	var branch string
	var err error

	if ref.Prev == "" {
		branch, err = GetDefaultBranch(DefaultRemote)
		if err != nil {
			return err
		}

	} else {
		branch = s.Refs[ref.Prev].Branch
	}

	err = CheckoutBranch(branch)
	if err != nil {
		return err
	}

	err = DeleteLocalBranch(ref.Branch)
	if err != nil {
		return err
	}

	return nil
}

func (s *Stack) adjustAdjacentRefs(ref StackRef) error {
	refs := s.Refs

	if ref.Prev != "" {
		prev := refs[ref.Prev]
		delete(refs, ref.Prev)

		prev.Next = ref.Next
		refs[ref.Prev] = prev

		err := UpdateStackRefFile(s.Title, prev)
		if err != nil {
			return fmt.Errorf("could not update reference file %v:", err)
		}
	}

	if ref.Next != "" {
		next := refs[ref.Next]
		delete(refs, ref.Next)

		next.Prev = ref.Prev
		refs[ref.Next] = next

		err := UpdateStackRefFile(s.Title, next)
		if err != nil {
			return fmt.Errorf("could not update reference file %v:", err)
		}
	}

	return nil
}

func (s *Stack) Last() (StackRef, error) {
	for _, ref := range s.Refs {
		if ref.Next == "" {
			return ref, nil
		}
	}

	return StackRef{}, fmt.Errorf("can't find the last ref in the chain. Data might be corrupted.")
}

func (s *Stack) First() (StackRef, error) {
	for _, ref := range s.Refs {
		if ref.Prev == "" {
			return ref, nil
		}
	}

	return StackRef{}, fmt.Errorf("can't find the first ref in the chain. Data might be corrupted.")
}

func GatherStackRefs(title string) (Stack, error) {
	stack := Stack{Title: title}
	stack.Refs = make(map[string]StackRef)

	root, err := StackRootDir(title)
	if err != nil {
		return stack, err
	}

	err = filepath.WalkDir(root, func(dir string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}
		// read files in the stacked ref directory
		// TODO: this may be quicker if we introduce a package
		// https://github.com/bmatcuk/doublestar
		if filepath.Ext(d.Name()) == ".json" {
			data, err := os.ReadFile(dir)
			if err != nil {
				return err
			}

			// marshal them into our StackRef type
			stackRef := StackRef{}
			err = json.Unmarshal(data, &stackRef)
			if err != nil {
				return err
			}

			stack.Refs[stackRef.SHA] = stackRef
		}

		return nil
	})
	if err != nil {
		if os.IsNotExist(err) { // there might not be any refs yet, this is ok.
			return stack, nil
		} else {
			return stack, err
		}
	}

	err = validateStackRefs(stack)
	if err != nil {
		return Stack{}, err
	}

	return stack, nil
}

func validateStackRefs(s Stack) error {
	endRefs := 0
	startRefs := 0

	for _, ref := range s.Refs {
		if ref.Next == "" {
			startRefs++
		}

		if ref.Prev == "" {
			endRefs++
		}

		if endRefs > 1 || startRefs > 1 {
			return errors.New("More than one end or start ref detected. Data might be corrupted.")
		}
	}
	return nil
}

func CurrentStackRefFromBranch(title string) (StackRef, error) {
	stack, err := GatherStackRefs(title)
	if err != nil {
		return StackRef{}, err
	}

	branch, err := CurrentBranch()
	if err != nil {
		return StackRef{}, err
	}

	for _, ref := range stack.Refs {
		if ref.Branch == branch {
			return ref, nil
		}
	}

	return StackRef{}, nil
}
