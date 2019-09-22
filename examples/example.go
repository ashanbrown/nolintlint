package examples

import "fmt"

func Foo() {
	fmt.Println("not specific")        //nolint
	fmt.Println("no explanation")      //nolint:my-linter
	fmt.Println("empty explanation")   //nolint:my-linter //
	fmt.Println("extra spaces")        //  nolint:my-linter // because
	fmt.Println("bad specific linter") // nolint: my-linter // because
}
