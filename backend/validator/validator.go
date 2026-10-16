package validator

import "regexp"

var (
	// ValidHashPattern проверяет MD5 хеш (32 символа hex)
	ValidHashPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

	// ValidEmotionPattern проверяет допустимые эмоции
	ValidEmotionPattern = regexp.MustCompile(`^(happy|sad|angry)$`)
)

// IsValidHash проверяет что строка является валидным MD5 хешем
func IsValidHash(hash string) bool {
	return ValidHashPattern.MatchString(hash)
}

// IsValidEmotion проверяет что строка является допустимой эмоцией
func IsValidEmotion(emotion string) bool {
	return ValidEmotionPattern.MatchString(emotion)
}
