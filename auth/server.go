package auth

import (
	"fmt"
	"net/http"
)

func WaitForAuthCode(expectedState string) (*UserAuth, error) {
	userAuthCh := make(chan *UserAuth)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		userAuth := &UserAuth{
			Code:  r.URL.Query().Get("code"),
			State: r.URL.Query().Get("state"),
			Error: r.URL.Query().Get("error"),
		}

		if userAuth.State != expectedState && userAuth.Error == "" {
			userAuth.Error = "Invalid state"
		}

		if userAuth.Error != "" {
			fmt.Fprintf(w, "Authorization failed %s", userAuth.Error)
		} else {
			fmt.Fprintf(w, "Authorization succeesful! You can close this window.")
		}

		userAuthCh <- userAuth
	})

	server := &http.Server{Addr: ":3000"}
	go func() {
		_ = server.ListenAndServe()
	}()

	userAuth := <-userAuthCh
	server.Close()

	return userAuth, nil
}
