package test

import (
	"fmt"
	"github.com/nuwa/bpp.v3/environment"
	"testing"
)

func TestA(t *testing.T) {
	fmt.Println(fmt.Sprint(false))
	fmt.Println(environment.Get("P_NAMESPACE_DEV"))
}
