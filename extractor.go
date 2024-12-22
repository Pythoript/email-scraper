package main

import (
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/robertkrimen/otto"
)

func GetEmails(htmlContent string, scrapedURL string) ([]string, error) {
	emails := make(map[string]struct{})

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	doc.Find("[data-cfemail]").Each(func(_ int, el *goquery.Selection) {
		cfemail, _ := el.Attr("data-cfemail")
		emails[deCFEmail(cfemail)] = struct{}{}
	})

	doc.Find("object[data], img[src], embed[src]").Each(func(_ int, el *goquery.Selection) {
		svgLink, exists := el.Attr("data")
		if !exists {
			svgLink, exists = el.Attr("src")
		}

		if exists && strings.HasSuffix(svgLink, ".svg") {
			parsedURL, err := url.Parse(svgLink)
			if err != nil {
				fmt.Println("Error parsing URL:", err)
				return
			}
			if !parsedURL.IsAbs() {
				baseURL, err := url.Parse(scrapedURL)
				if err != nil {
					fmt.Println("Error parsing base URL:", err)
					return
				}
				parsedURL = baseURL.ResolveReference(parsedURL)
			}

			absoluteSVGLink := parsedURL.String()
			svgEmails, err := extractEmailsFromSVG(absoluteSVGLink)
			if err == nil {
				for _, email := range svgEmails {
					emails[email] = struct{}{}
				}
			}
		}
	})

	extractedEmails := extractEmails(htmlContent)
	for _, email := range extractedEmails {
		emails[email] = struct{}{}
	}

	doc.Find("a").Each(func(_ int, el *goquery.Selection) {
		href, _ := el.Attr("href")
		hrefEmails := extractEmailsFromHref(href)
		for _, email := range hrefEmails {
			emails[email] = struct{}{}
		}
	})

	var result []string
	for email := range emails {
		result = append(result, email)
	}
	return result, nil
}

func deCFEmail(encodedString string) string {
	r, _ := hex.DecodeString(encodedString[:2])
	decoded := make([]byte, len(encodedString)/2-1)
	for i := 2; i < len(encodedString); i += 2 {
		val, _ := hex.DecodeString(encodedString[i : i+2])
		decoded[(i/2)-1] = val[0] ^ r[0]
	}
	return string(decoded)
}

func rotIsValid(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.(com|edu|net|ca|co.uk|co|org|gov|info|de|au|nl|ru|fr|br|uk|io|me|dev)+$`)
	return emailRegex.MatchString(email)
}

func rotateChar(char rune, x int) rune {
	switch {
	case unicode.IsLower(char):
		return 'a' + (char-'a'+rune(x))%26
	case unicode.IsUpper(char):
		return 'A' + (char-'A'+rune(x))%26
	default:
		return char
	}
}

func tryRotDecryption(encryptedEmail string) []string {
	var potentialEmails []string
	for x := 1; x < 26; x++ {
		decrypted := strings.Map(func(r rune) rune { return rotateChar(r, x) }, encryptedEmail)
		if rotIsValid(decrypted) {
			potentialEmails = append(potentialEmails, decrypted)
		}
	}
	return potentialEmails
}

func normalizeHTML(htmlStr string) string {
	decoded, err := url.QueryUnescape(htmlStr)
	if err != nil {
		decoded = htmlStr
	}
	decoded = html.UnescapeString(decoded)
	decoded = decodeUnicodeEscape(decoded)
	return decoded
}

func decodeUnicodeEscape(str string) string {
	unicodeEscapePattern := regexp.MustCompile(`\\u[0-9a-fA-F]{4}`)
	return unicodeEscapePattern.ReplaceAllStringFunc(str, func(match string) string {
		r, _ := hex.DecodeString(match[2:])
		return string(r)
	})
}

func processHTML(text string) string {
	text = normalizeHTML(text)
	text += " " + strings.Join(tryRotDecryption(text), " ")
	text = regexp.MustCompile(`<!--(.|\s|\n)*?-->`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, "")
	return text
}

func preProcessEmail(text string) string {
	text = processHTML(text)
	text = regexp.MustCompile(`\W*\.\W*|\W+(D|d)(O|0|o)(t|T)\W+`).ReplaceAllString(text, ".")
	text = regexp.MustCompile(`([a-z0-9])(DOT|D0T|DoT)([a-z0-9])`).ReplaceAllString(text, "$1.$3")
	text = regexp.MustCompile(`([A-Z0-9])(dot|d0t|dOt)([A-Z0-9])`).ReplaceAllString(text, "$1.$3")
	text = regexp.MustCompile(`\W*@\W*|\W+(A|a)(T|t)\W+`).ReplaceAllString(text, "@")
	text = regexp.MustCompile(`([a-z0-9])AT([a-z0-9])`).ReplaceAllString(text, "$1@$2")
	text = regexp.MustCompile(`([A-Z0-9])at([A-Z0-9])`).ReplaceAllString(text, "$1@$2")
	text = regexp.MustCompile(`([a-z0-9])REMOVE([a-z0-9@\.])`).ReplaceAllString(text, "$1$2")
	text = regexp.MustCompile(`([A-Z0-9])remove([A-Z0-9@\.])`).ReplaceAllString(text, "$1$2")
	text = regexp.MustCompile(`[_\W]*n[_\W]*(o|0)[_\W]*(s|5)[_\W]*p[_\W]*a[_\W]*m[_\W]*`).ReplaceAllStringFunc(text, func(match string) string {
		if strings.Contains(match, ".") {
			return "."
		} else if strings.Contains(match, "@") {
			return "@"
		}
		return ""
	})
	return text
}

func evaluateJSExpression(expression string) string {
	vm := otto.New()
	value, err := vm.Run(expression)
	if err != nil {
		return ""
	}
	result, err := value.ToString()
	if err != nil {
		return ""
	}
	return result
}

func extractEmailsFromHref(href string) []string {
	if strings.HasPrefix(href, "javascript:") {
		expression := strings.TrimPrefix(href, "javascript:")
		decoded := evaluateJSExpression(expression)
		return extractEmails(decoded)
	}
	return extractEmails(href)
}

func extractEmails(htmlString string) []string {
	cleanedString := preProcessEmail(htmlString)
	emailRegex := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[a-zA-Z]{2,}\b`)
	rtlPattern := regexp.MustCompile(`\b(moc|ac|gro|ten|ppa|due|ku\.oc)\.[a-zA-Z0-9.-]+@[A-Za-z0-9._%+-]+\b`)

	rtlEmails := rtlPattern.FindAllString(cleanedString, -1)
	for i := range rtlEmails {
		rtlEmails[i] = reverseString(rtlEmails[i])
	}

	foundEmails := emailRegex.FindAllString(cleanedString, -1)
	allExtractedEmails := append(foundEmails, rtlEmails...)
	return allExtractedEmails
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func fetchSVGContent(svgURL string) (string, error) {
	resp, err := http.Get(svgURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	return doc.Text(), nil
}

func decodeHexAndUnicode(input string) string {
	re := regexp.MustCompile(`&#x([0-9a-fA-F]+);`)
	return re.ReplaceAllStringFunc(input, func(hexStr string) string {
		code, _ := hex.DecodeString(hexStr[3 : len(hexStr)-1])
		return string(code)
	})
}

func extractEmailsFromSVG(svgURL string) ([]string, error) {
	svgContent, err := fetchSVGContent(svgURL)
	if err != nil {
		return nil, err
	}

	decodedContent := decodeHexAndUnicode(svgContent)
	return extractEmails(decodedContent), nil
}
