//go:build windows

package packages

func MapResolverVersion(resolver LatestResolver, fn func(string) string) LatestResolver {
	return func(log func(string)) (url, fileName, stripTop, version string, err error) {
		url, fileName, stripTop, version, err = resolver(log)
		if err != nil {
			return "", "", "", "", err
		}
		return url, fileName, stripTop, fn(version), nil
	}
}
