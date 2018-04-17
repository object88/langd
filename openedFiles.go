package langd

import (
	"fmt"

	"github.com/object88/rope"
)

type OpenedFiles struct {
	ropes map[string]*rope.Rope
}

func (of *OpenedFiles) EnsureOpened(absFilepath, text string) error {
	if _, ok := of.ropes[absFilepath]; ok {
		return fmt.Errorf("File %s is already opened", absFilepath)
	}
	of.ropes[absFilepath] = rope.CreateRope(text)
	return nil
}

func (of *OpenedFiles) Close(absFilepath string) error {
	_, ok := of.ropes[absFilepath]
	if !ok {
		return fmt.Errorf("File %s is not opened", absFilepath)
	}

	delete(of.ropes, absFilepath)
	return nil
}

func (of *OpenedFiles) Get(absFilepath string) (*rope.Rope, error) {
	buf, ok := of.ropes[absFilepath]
	if !ok {
		return nil, fmt.Errorf("File %s is not opened", absFilepath)
	}

	return buf, nil
}

func (of *OpenedFiles) Replace(absFilepath, text string) error {
	_, ok := of.ropes[absFilepath]
	if !ok {
		return fmt.Errorf("File %s is not opened", absFilepath)
	}

	// Replace the entire document
	buf := rope.CreateRope(text)
	of.ropes[absFilepath] = buf

	return nil
}
