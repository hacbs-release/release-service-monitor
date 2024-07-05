package checks

type QuayAuth struct {
    username string
    password string
}

func NewQuayAuth(username string, password string) (*QuayAuth) {
    auth := &QuayAuth{
        username: username,
        password: password,
    }

    return auth
}

func (a *QuayAuth) getUsername() (string) {
    return a.username
}

func (a *QuayAuth) getPassword() (string) {
    return a.password
}
