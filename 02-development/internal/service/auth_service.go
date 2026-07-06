package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"pottery-api/internal/domain"
	"pottery-api/internal/repository"
	"pottery-api/internal/service/auth"
)

type AuthService struct {
	otpStore   repository.OTPStore
	clientRepo repository.ClientStore
	jwt        *auth.JWTService
}

func NewAuthService(otp repository.OTPStore, clients repository.ClientStore, jwt *auth.JWTService) *AuthService {
	return &AuthService{
		otpStore:   otp,
		clientRepo: clients,
		jwt:        jwt,
	}
}

func (s *AuthService) SendOTP(ctx context.Context, phone string) (ttl int, err error) {
	// Rate limit: 1 send per 60 seconds
	sendCount, err := s.otpStore.IncrSendCount(ctx, phone, 60)
	if err != nil {
		return 0, err
	}
	if sendCount > 1 {
		return 0, domain.ErrTooManyRequests
	}

	// Brute-force block: 5 fails per hour → block 30 min
	failCount, err := s.otpStore.IncrFailCount(ctx, phone, 3600)
	if err != nil {
		return 0, err
	}
	if failCount > 5 {
		return 0, domain.ErrTooManyRequests
	}

	code, err := generateOTPCode()
	if err != nil {
		return 0, err
	}

	if err := s.otpStore.Set(ctx, phone, code, 300); err != nil {
		return 0, err
	}
	return 300, nil
}

func (s *AuthService) VerifyOTP(ctx context.Context, phone, code, name string) (*auth.TokenPair, *domain.UserProfile, error) {
	storedCode, err := s.otpStore.Get(ctx, phone)
	if err != nil {
		return nil, nil, domain.ErrInvalidOTP
	}

	attempts, err := s.otpStore.IncrAttempts(ctx, phone, 300)
	if err != nil {
		return nil, nil, err
	}
	if attempts > 3 {
		return nil, nil, domain.ErrTooManyRequests
	}

	if storedCode != code {
		return nil, nil, domain.ErrInvalidOTP
	}

	client, err := s.clientRepo.GetByPhone(ctx, phone)
	if err != nil {
		if err == domain.ErrNotFound {
			client, err = s.clientRepo.Create(ctx, name, phone)
			if err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	}

	tokens, err := s.jwt.Generate(client.ID, client.Phone)
	if err != nil {
		return nil, nil, err
	}

	_ = s.otpStore.Delete(ctx, phone)

	return tokens, &domain.UserProfile{
		ID:    client.ID,
		Name:  client.Name,
		Phone: client.Phone,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	claims, err := s.jwt.Validate(refreshToken)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	client, err := s.clientRepo.GetByID(ctx, claims.Subject)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	return s.jwt.Generate(client.ID, client.Phone)
}

func generateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}