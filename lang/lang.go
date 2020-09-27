package lang

import (
	"fmt"
	"regexp"
	"strings"
)

// ToSentence splits up camel cased function names
func ToSentence(name string) []string {
	re := regexp.MustCompile(`(([^A-Z]+)|([A-Z]{2})|([A-Z][^A-Z]+))`)
	return re.FindAllString(name, -1)
}

func eToIng(verb string) string {
	return fmt.Sprintf("%sing", strings.TrimSuffix(verb, "e"))

}

func eToEd(verb string) string {
	if strings.HasSuffix(verb, "e") {
		return fmt.Sprintf("%sd", verb)
	}
	return fmt.Sprintf("%sed", verb)
}

func ToPresentParticiple(words []string) string {
	if len(words) > 1 {
		return fmt.Sprintf("%s %s", eToIng(words[0]), strings.Join(words[1:], " "))
	}

	return eToIng(words[0])
}

func VerbToPastTense(verb string) string {
	if verb == "build" {
		return "built"
	}
	return eToEd(verb)
}

func ToPastTensePhrase(words []string) string {
	return fmt.Sprintf("%s %s", VerbToPastTense(words[0]), strings.Join(words[1:], " "))
}
