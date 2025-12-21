package source

func IsPublicBind(addrs []string) bool {
	for _, a := range addrs {
		if a == "0.0.0.0" || a == "::" {
			return true
		}
	}
	return false
}
