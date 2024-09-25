package git

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
)

type StackRef struct {
	Prev        string `json:"prev"`
	Branch      string `json:"branch"`
	SHA         string `json:"sha"`
	Next        string `json:"next"`
	MR          string `json:"mr"`
	Description string `json:"description"`
}

// Stack represents a stacked diff data structure.
// Refs are structured as a doubly linked list where
// the links are identified with the StackRef.Prev
// and StackRef.Next fields.
// The StackRef.SHA is the id that the former two
// fields can point to.
// All stacks must be created with GatherStackRefs
// which validates the stack for consistency.
type Stack struct {
	Title string
	Refs  map[string]StackRef
}

func (s Stack) Empty() bool { return len(s.Refs) == 0 }

func (s *Stack) RemoveRef(ref StackRef) error {
	if ref.IsFirst() && ref.IsLast() {
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

	if ref.IsFirst() {
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

func (s *Stack) Last() StackRef {
	if s.Empty() {
		return StackRef{}
	}

	for _, ref := range s.Refs {
		if ref.IsLast() {
			return ref
		}
	}

	// All Stacks should be created with GatherStackRefs which validates the Stack consistency.
	panic(errors.New("can't find the last ref in the chain. Data might be corrupted."))
}

func (s *Stack) First() StackRef {
	if s.Empty() {
		return StackRef{}
	}

	for _, ref := range s.Refs {
		if ref.IsFirst() {
			return ref
		}
	}

	// All Stacks should be created with GatherStackRefs which validates the Stack consistency.
	panic(errors.New("can't find the first ref in the chain. Data might be corrupted."))
}

// Iter returns an iterator to range from the first to the last ref in the stack.
func (s *Stack) Iter() iter.Seq[StackRef] {
	return func(yield func(StackRef) bool) {
		ref := s.First()
		for !ref.Empty() {
			if !yield(ref) {
				return
			}

			ref = s.Refs[ref.Next]
		}
	}
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

	if s.Empty() {
		// empty stacks are okay.
		return nil
	}

	for _, ref := range s.Refs {
		if ref.IsFirst() {
			startRefs++
		}

		if ref.IsLast() {
			endRefs++
		}

		if endRefs > 1 || startRefs > 1 {
			return errors.New("More than one end or start ref detected. Data might be corrupted.")
		}
	}

	if startRefs != 1 {
		return errors.New("expected exactly one start ref. Data might be corrupted.")
	}
	if endRefs != 1 {
		return errors.New("expected exactly one end ref. Data might be corrupted.")
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

// Empty returns true if the stack ref does not have an associated SHA (commit).
// This indicates that the StackRef is invalid.
func (r StackRef) Empty() bool { return r.SHA == "" }

// IsFirst returns true if the stack ref is the first of the stack.
// A stack ref is considered the first if it does not reference any previous ref.
func (r StackRef) IsFirst() bool { return r.Prev == "" }

// IsLast returns true if the stack ref is the last of the stack.
// A stack ref is considered the last if it does not reference any next ref.
func (r StackRef) IsLast() bool { return r.Next == "" }

// Subject returns the stack ref description suitable as commit Subject
// and for other in space limited places.
// It only takes the first line of the description into account
// and truncates it to 72 characters.
func (r StackRef) Subject() string {
	ls := strings.SplitN(r.Description, "\n", 1)
	if len(ls[0]) <= 72 {
		return ls[0]
	}

	return ls[0][:69] + "..."
}
