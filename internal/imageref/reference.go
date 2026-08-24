package imageref

import (
	"fmt"
	"strings"

	ref "github.com/distribution/reference"
)

type TaggedReference struct {
	Normalized string
	Repository string
	Familiar   string
	Tag        string
}

func ParseTagged(value string) (TaggedReference, error) {
	parsed, err := ref.ParseDockerRef(strings.TrimSpace(value))
	if err != nil {
		return TaggedReference{}, err
	}
	tagged, ok := parsed.(ref.NamedTagged)
	if !ok {
		return TaggedReference{}, fmt.Errorf("镜像引用 %q 没有 tag", value)
	}
	trimmed := ref.TrimNamed(tagged)
	return TaggedReference{
		Normalized: tagged.String(),
		Repository: trimmed.Name(),
		Familiar:   ref.FamiliarName(trimmed),
		Tag:        tagged.Tag(),
	}, nil
}

func CacheKey(value string) string {
	parsed, err := ParseTagged(value)
	if err != nil {
		return strings.TrimSpace(value)
	}
	return parsed.Normalized
}
