package service

import (
	"context"
	"kenalbatik-be/internal/domain"
	quizRepo "kenalbatik-be/internal/quiz/repository"
	userRepo "kenalbatik-be/internal/user/repository"
	userAnswerRepo "kenalbatik-be/internal/userAnswer/repository"
	"time"

	"github.com/google/uuid"
)

type QuizService interface {
	GetQuizzes(ctx context.Context, userId uuid.UUID) ([]domain.Quiz, error)
	CheckAnswer(ctx context.Context, userId uuid.UUID, userAnswer domain.AnswerRequest) (domain.AnswerResponse, error)
}

type quizService struct {
	userRepo       userRepo.Userepository
	quizRepo       quizRepo.QuizRepository
	userAnswerRepo userAnswerRepo.UserAnswerRepositoryInterface
}

func NewQuizService(userRepo userRepo.Userepository, quizRepo quizRepo.QuizRepository, userAnswerRepo userAnswerRepo.UserAnswerRepositoryInterface) QuizService {
	return &quizService{
		userRepo:       userRepo,
		quizRepo:       quizRepo,
		userAnswerRepo: userAnswerRepo,
	}
}

func (s *quizService) GetQuizzes(ctx context.Context, userId uuid.UUID) ([]domain.Quiz, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var user domain.User
	err := s.userRepo.FindUser(ctx, &user, domain.UserParam{ID: userId})
	if err != nil {
		return nil, err
	}

	var quizzes []domain.Quiz

	switch user.Tier {
	case domain.TIER1:
		err = s.quizRepo.GetQuiz(ctx, &quizzes, "DIFFICULTY = ?", []interface{}{domain.EASY})
	case domain.TIER2:
		err = s.quizRepo.GetQuiz(ctx, &quizzes, "DIFFICULTY = ? OR DIFFICULTY = ?", []interface{}{domain.EASY, domain.MEDIUM})
	case domain.TIER3:
		err = s.quizRepo.GetQuiz(ctx, &quizzes, "DIFFICULTY = ?", []interface{}{domain.MEDIUM})
	case domain.TIER4:
		err = s.quizRepo.GetQuiz(ctx, &quizzes, "DIFFICULTY = ? OR DIFFICULTY = ?", []interface{}{domain.MEDIUM, domain.HARD})
	case domain.TIER5:
		err = s.quizRepo.GetQuiz(ctx, &quizzes, "DIFFICULTY = ?", []interface{}{domain.HARD})
	}
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, domain.ErrTimeout
	default:
		return quizzes, nil
	}
}

func (s *quizService) CheckAnswer(ctx context.Context, userId uuid.UUID, userAnswer domain.AnswerRequest) (domain.AnswerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var user domain.User
	err := s.userRepo.FindUser(ctx, &user, domain.UserParam{ID: userId})
	if err != nil {
		return domain.AnswerResponse{}, err
	}

	var correctAnswer int

	for i, quizId := range userAnswer.QuizID {
		var quiz domain.Quiz
		err := s.quizRepo.GetQuizByID(ctx, &quiz, quizId)
		if err != nil {
			return domain.AnswerResponse{}, err
		}

		if quiz.Answer == userAnswer.UserAnswer[i] {
			correctAnswer++

			switch quiz.Difficulty {
			case domain.EASY:
				user.Experience += 3
			case domain.MEDIUM:
				user.Experience += 5
			case domain.HARD:
				user.Experience += 7
			}
		}
	}

	if user.Experience >= 100 {
		user.Tier = domain.TIER5
	} else if user.Experience >= 75 && user.Experience < 100 {
		user.Tier = domain.TIER4
	} else if user.Experience >= 50 && user.Experience < 70 {
		user.Tier = domain.TIER3
	} else if user.Experience >= 25 && user.Experience < 50 {
		user.Tier = domain.TIER2
	} else {
		user.Tier = domain.TIER1
	}

	userAnswerData := domain.UserAnswer{
		UserID:             userId,
		QuizID:             userAnswer.QuizID,
		UserAnswer:         userAnswer.UserAnswer,
		TotalCorrectAnswer: correctAnswer,
	}

	err = s.userAnswerRepo.Create(ctx, &userAnswerData)
	if err != nil {
		return domain.AnswerResponse{}, err
	}

	err = s.userRepo.UpdateUser(ctx, user, domain.UserParam{ID: userId})
	if err != nil {
		return domain.AnswerResponse{}, err
	}

	res := domain.AnswerResponse{
		CorrectAnswer:  correctAnswer,
		UserExperience: user.Experience,
		UserTier:       user.Tier,
	}

	select {
	case <-ctx.Done():
		return domain.AnswerResponse{}, domain.ErrTimeout
	default:
		return res, nil
	}
}