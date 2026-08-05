package xutil

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var stripTagRegx = regexp.MustCompile(`<(.|\n)*?>`)

func StripTags(content string) string {
	return stripTagRegx.ReplaceAllString(content, "")
}

func Md5(str string) string {
	h := md5.New()
	_, err := io.WriteString(h, str)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func MaskEmail(i string) string {
	l := len(i)
	if l == 0 {
		return ""
	}

	tmp := strings.Split(i, "@")
	if len(tmp) == 1 {
		return MaskString(i)
	}

	addr := tmp[0]
	domain := tmp[1]

	return MaskString(addr) + "@" + domain
}

func MaskString(s string) string {
	list := strings.Split(s, "")
	for i, s2 := range list {
		if i%2 == 1 {
			list[i] = "*"
		} else {
			list[i] = s2
		}
	}
	return strings.Join(list, "")
}

func SubString(s string, start, length int) string {
	r := []rune(s)
	if len(r) <= length {
		return s
	}
	return string(r[start : length+start])
}

func SplitByWidth(str string, size int) []string {
	chars := []rune(str)
	strLength := len(chars)
	var splited []string
	var stop int
	for i := 0; i < strLength; i += size {
		stop = i + size
		if stop > strLength {
			stop = strLength
		}
		splited = append(splited, string(chars[i:stop]))
	}
	return splited
}

func RandomString(length int) string {
	b := make([]byte, length/2)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
