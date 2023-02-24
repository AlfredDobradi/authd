package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"golang.org/x/crypto/bcrypt"
)

const cost int = bcrypt.DefaultCost

var CLI struct {
	Generate GenerateCmd `cmd:"" help:"Generate a username-hash pair"`

	Test TestCmd `cmd:"" help:"List paths."`
}

type GenerateCmd struct {
	Username string
	Password string
}

func (g *GenerateCmd) Run() error {
	if g.Username == "" {
		return fmt.Errorf("Username can't be empty")
	}

	if g.Password == "" {
		return fmt.Errorf("Password can't be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(g.Password), cost)
	if err != nil {
		return fmt.Errorf("Failed to generate hash: %w", err)
	}

	fmt.Printf("\"%s\": \"%s\"\n", g.Username, string(hash))
	return nil
}

type TestCmd struct {
	Password string
	Hash     string
}

func (t *TestCmd) Run() error {
	if t.Password == "" {
		return fmt.Errorf("Password can't be empty")
	}

	if t.Hash == "" {
		return fmt.Errorf("Hash can't be empty")
	}

	err := bcrypt.CompareHashAndPassword([]byte(t.Hash), []byte(t.Password))
	if err != nil {
		return fmt.Errorf("Failed to compare hash: %w", err)
	}

	fmt.Println("Password is valid")
	return nil
}

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
