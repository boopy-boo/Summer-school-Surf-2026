package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type OTPStore struct {
	client *redis.Client
}

func NewOTPStore(addr, password string) *OTPStore {
	return &OTPStore{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       0,
		}),
	}
}

func (s *OTPStore) key(phone string) string {
	return fmt.Sprintf("otp:%s", phone)
}

func (s *OTPStore) attemptsKey(phone string) string {
	return fmt.Sprintf("otp:%s:attempts", phone)
}

func (s *OTPStore) sendKey(phone string) string {
	return fmt.Sprintf("otp:%s:send", phone)
}

func (s *OTPStore) failKey(phone string) string {
	return fmt.Sprintf("otp:%s:fail", phone)
}

func (s *OTPStore) Set(ctx context.Context, phone, code string, ttl int) error {
	return s.client.Set(ctx, s.key(phone), code, time.Duration(ttl)*time.Second).Err()
}

func (s *OTPStore) Get(ctx context.Context, phone string) (string, error) {
	return s.client.Get(ctx, s.key(phone)).Result()
}

func (s *OTPStore) IncrAttempts(ctx context.Context, phone string, ttl int) (int, error) {
	k := s.attemptsKey(phone)
	pipe := s.client.Pipeline()
	incr := pipe.Incr(ctx, k)
	pipe.Expire(ctx, k, time.Duration(ttl)*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int(incr.Val()), nil
}

func (s *OTPStore) IncrSendCount(ctx context.Context, phone string, window int) (int, error) {
	k := s.sendKey(phone)
	pipe := s.client.Pipeline()
	incr := pipe.Incr(ctx, k)
	pipe.Expire(ctx, k, time.Duration(window)*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int(incr.Val()), nil
}

func (s *OTPStore) IncrFailCount(ctx context.Context, phone string, window int) (int, error) {
	k := s.failKey(phone)
	pipe := s.client.Pipeline()
	incr := pipe.Incr(ctx, k)
	pipe.Expire(ctx, k, time.Duration(window)*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int(incr.Val()), nil
}

func (s *OTPStore) Delete(ctx context.Context, phone string) error {
	return s.client.Del(ctx, s.key(phone), s.attemptsKey(phone), s.sendKey(phone)).Err()
}