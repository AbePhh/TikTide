package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword 使用 bcrypt 对明文密码进行哈希。
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ComparePassword 校验哈希密码与明文密码是否匹配。
func ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
