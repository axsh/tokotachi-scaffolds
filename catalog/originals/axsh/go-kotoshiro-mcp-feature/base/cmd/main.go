package main

import (
	"function/internal/function"

	"github.com/axsh/kuniumi"
)

func main() {
	app := kuniumi.New(kuniumi.Config{Name: "TemplateFunc", Version: "0.1.0"})
	app.RegisterFunc(function.Add, "Adds two integers",
		kuniumi.WithParams(
			kuniumi.Param("x", "First integer"),
			kuniumi.Param("y", "Second integer"),
		),
		kuniumi.WithReturns("Sum of x and y"),
	)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
