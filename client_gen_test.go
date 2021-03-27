package main

import (
	"fmt"
	"testing"
)

func TestGenerateClient(t *testing.T) {
	client := Client{
		Name: "auth",
		funcs: []Func{
			Func{
				Name: "login",
				params: []Param{
					{
						Name: "UserLogin",
					},
				},
				Resp: "LoginResp",
			},
		},
	}
	s := generateClient(client)
	fmt.Println(s)
}
