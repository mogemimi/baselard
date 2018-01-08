package main

import (
	"bufio"
	"os"
)

type MSBuildGenerator struct {
}

func (gen *MSBuildGenerator) WriteFile(ninjaFile string) error {
	file, err := os.Create(ninjaFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	writer.Flush()

	return nil
}
