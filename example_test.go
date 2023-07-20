package falta_test

import (
	"fmt"

	"github.com/a20r/falta"
)

type Circle struct {
	Radius float64
}

var ErrInvalidCircle = falta.New[Circle]("invalid circle: radius ({{.Radius}}) <= 0")

func IsCircleValid(circle Circle) error {
	if circle.Radius <= 0 {
		return ErrInvalidCircle.New(circle)
	}

	return nil
}

func ExampleFactory() {
	circle := Circle{Radius: -1}

	if err := IsCircleValid(circle); err != nil {
		fmt.Println(err)
	}

	// Output:
	// invalid circle: radius (-1) <= 0
}
