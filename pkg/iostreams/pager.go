package iostreams

import "os"

func PagerCommandFromEnv() string {
	if glabPager, glabPagerExists := os.LookupEnv("GLAB_PAGER"); glabPagerExists {
		return glabPager
	} else {
		return os.Getenv("PAGER")
	}
}
