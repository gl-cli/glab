package prompt

import (
	"fmt"
	"reflect"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
)

func StubConfirm(result bool) func() {
	orig := Confirm
	Confirm = func(r *bool, _ string, _ bool) error {
		*r = result
		return nil
	}
	return func() {
		Confirm = orig
	}
}

type AskStubber struct {
	Asks     [][]*survey.Question
	AskOnes  []*survey.Prompt
	Count    int
	OneCount int
	Stubs    [][]*QuestionStub
	StubOnes []*PromptStub
}

func InitAskStubber() (*AskStubber, func()) {
	origAsk := Ask
	origAskOne := AskOne
	as := AskStubber{}

	AskOne = func(p survey.Prompt, response any, opts ...survey.AskOpt) error {
		as.AskOnes = append(as.AskOnes, &p)
		count := as.OneCount
		as.OneCount += 1
		if count >= len(as.StubOnes) {
			panic(fmt.Sprintf("more asks than stubs. most recent call: %v", p))
		}
		stubbedPrompt := as.StubOnes[count]
		if stubbedPrompt.Default {
			// TODO this is failing for basic AskOne invocations with a string result.
			defaultValue := reflect.ValueOf(p).Elem().FieldByName("Default")

			// If the user passed us an error, return it
			if err, ok := defaultValue.Interface().(error); ok {
				return err
			}

			_ = core.WriteAnswer(response, "", defaultValue)
		} else {
			// If the user passed us an error, return it
			if err, ok := stubbedPrompt.Value.(error); ok {
				return err
			}

			_ = core.WriteAnswer(response, "", stubbedPrompt.Value)
		}

		return nil
	}

	Ask = func(qs []*survey.Question, response any, opts ...survey.AskOpt) error {
		as.Asks = append(as.Asks, qs)
		count := as.Count
		as.Count += 1
		if count >= len(as.Stubs) {
			panic(fmt.Sprintf("more asks than stubs. most recent call: %#v", qs))
		}

		// actually set response
		stubbedQuestions := as.Stubs[count]
		for i, sq := range stubbedQuestions {
			q := qs[i]
			if q.Name != sq.Name {
				panic(fmt.Sprintf("stubbed question mismatch: %s != %s", q.Name, sq.Name))
			}
			if sq.Default {
				defaultValue := reflect.ValueOf(q.Prompt).Elem().FieldByName("Default")

				// If the user passed us an error, return it
				if err, ok := defaultValue.Interface().(error); ok {
					return err
				}

				_ = core.WriteAnswer(response, q.Name, defaultValue)
			} else {
				// If the user passed us an error, return it
				if err, ok := sq.Value.(error); ok {
					return err
				}

				_ = core.WriteAnswer(response, q.Name, sq.Value)
			}
		}

		return nil
	}
	teardown := func() {
		Ask = origAsk
		AskOne = origAskOne
	}
	return &as, teardown
}

type PromptStub struct {
	Value   any
	Default bool
}

type QuestionStub struct {
	Name    string
	Value   any
	Default bool
}

func (as *AskStubber) StubOne(value any) {
	as.StubOnes = append(as.StubOnes, &PromptStub{
		Value: value,
	})
}

func (as *AskStubber) Stub(stubbedQuestions []*QuestionStub) {
	// A call to .Ask takes a list of questions; a stub is then a list of questions in the same order.
	as.Stubs = append(as.Stubs, stubbedQuestions)
}
