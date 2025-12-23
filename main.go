package main

import (
	"fmt"

	"github.com/SrVariable/SPL/auth"
	"github.com/SrVariable/SPL/config"
	"github.com/SrVariable/SPL/tools"
)

func main() {
	env, err := config.NewEnv()
	if err != nil {
		return
	}

	state, err := tools.GenerateCode(60)
	if err != nil {
		return
	}

	codeVerifier, err := tools.GenerateCodeVerifier()
	if err != nil {
		return
	}

	codeChallenge := tools.GenerateCodeChallenge(codeVerifier)

	userAuthParams := auth.UserAuthParams{
		ClientID:            env.ClientId,
		ResponseType:        "code",
		RedirectURI:         env.RedirectURI,
		State:               state,
		Scope:               "user-read-private user-read-email user-read-playback-state playlist-modify-public playlist-modify-private playlist-read-private",
		CodeChallengeMethod: "S256",
		CodeChallenge:       codeChallenge,
	}

	accessToken, err := auth.GetAccessToken(codeVerifier, userAuthParams)
	if err != nil {
		fmt.Println(err)
		return
	}

	accessToken.Save()
}
