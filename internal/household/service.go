package household

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/AliFnieer/needly-backend/internal/notification"
	"gorm.io/gorm"
)

// Service handles household business logic.
type Service struct {
	db           *gorm.DB
	notification *notification.Service
}

// CreateRequest is the payload for creating a household.
type CreateRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

// UpdateRequest is the payload for updating a household.
type UpdateRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

// AddMemberRequest is the payload for adding a member to a household.
type AddMemberRequest struct {
	UserID uint `json:"user_id" binding:"required"`
	Role   Role `json:"role" binding:"omitempty,oneof=owner member"`
}

// NewService creates a new household service.
func NewService(db *gorm.DB, notificationSvc *notification.Service) *Service {
	return &Service{
		db:           db,
		notification: notificationSvc,
	}
}

// Create creates a new household and adds the creator as owner.
func (s *Service) Create(ownerID uint, req *CreateRequest) (*Household, error) {
	household := Household{
		Name:    req.Name,
		OwnerID: ownerID,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Create the household
		if err := tx.Create(&household).Error; err != nil {
			return err
		}

		// Add the creator as owner member
		member := HouseholdMember{
			HouseholdID: household.ID,
			UserID:      ownerID,
			Role:        RoleOwner,
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create household: %w", err)
	}

	// Notify the household members about the new household
	s.notify(context.Background(), notification.NotificationTypeHouseholdCreated,
		"Household created",
		fmt.Sprintf("Household %q was created", household.Name),
		household.ID, 0, 0, ownerID)

	return &household, nil
}

// GetByID retrieves a household by ID.
func (s *Service) GetByID(id uint) (*Household, error) {
	var household Household
	if err := s.db.Preload("Members").First(&household, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("household not found")
		}
		return nil, fmt.Errorf("failed to get household: %w", err)
	}
	return &household, nil
}

// ListByUser retrieves all households the user is a member of.
func (s *Service) ListByUser(userID uint) ([]Household, error) {
	var households []Household
	if err := s.db.
		Preload("Members").
		Joins("JOIN household_members ON household_members.household_id = households.id").
		Where("household_members.user_id = ?", userID).
		Find(&households).Error; err != nil {
		return nil, fmt.Errorf("failed to list households for user: %w", err)
	}

	return households, nil
}

// Update updates household details. Only the owner can update.
func (s *Service) Update(id, userID uint, req *UpdateRequest) (*Household, error) {
	if err := s.checkOwner(id, userID); err != nil {
		return nil, err
	}

	var household Household
	if err := s.db.First(&household, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("household not found")
		}
		return nil, fmt.Errorf("failed to get household: %w", err)
	}

	household.Name = req.Name
	if err := s.db.Save(&household).Error; err != nil {
		return nil, fmt.Errorf("failed to update household: %w", err)
	}

	// Notify the household members about the update
	s.notify(context.Background(), notification.NotificationTypeHouseholdUpdated,
		"Household updated",
		fmt.Sprintf("Household %q was updated", household.Name),
		household.ID, 0, 0, userID)

	return &household, nil
}

// Delete removes a household. Only the owner can delete.
func (s *Service) Delete(id, userID uint) error {
	if err := s.checkOwner(id, userID); err != nil {
		return err
	}

	var household Household
	if err := s.db.First(&household, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("household not found")
		}
		return fmt.Errorf("failed to get household: %w", err)
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Delete all members
		if err := tx.Where("household_id = ?", id).Delete(&HouseholdMember{}).Error; err != nil {
			return err
		}
		// Delete the household
		if err := tx.Delete(&household).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to delete household: %w", err)
	}

	// Notify members about the household deletion
	s.notify(context.Background(), notification.NotificationTypeHouseholdDeleted,
		"Household deleted",
		fmt.Sprintf("Household %q was deleted", household.Name),
		household.ID, 0, 0, userID)

	return nil
}

// AddMember adds a user to a household. Only the owner can add members.
func (s *Service) AddMember(id, ownerID uint, req *AddMemberRequest) (*HouseholdMember, error) {
	if err := s.checkOwner(id, ownerID); err != nil {
		return nil, err
	}

	role := req.Role
	if role == "" {
		role = RoleMember
	}

	member := HouseholdMember{
		HouseholdID: id,
		UserID:      req.UserID,
		Role:        role,
	}

	if err := s.db.Create(&member).Error; err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	// Notify the household members about the added member
	var household Household
	if err := s.db.First(&household, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get household: %w", err)
	}

	s.notify(context.Background(), notification.NotificationTypeMemberAdded,
		"New household member",
		fmt.Sprintf("A new member was added to household %q", household.Name),
		household.ID, 0, 0, ownerID)

	return &member, nil
}

// RemoveMember removes a user from a household. Only the owner can remove members.
func (s *Service) RemoveMember(id, ownerID, memberUserID uint) error {
	if err := s.checkOwner(id, ownerID); err != nil {
		return err
	}

	// Prevent removing the owner
	if memberUserID == ownerID {
		return errors.New("owner cannot be removed from their own household")
	}

	result := s.db.Where("household_id = ? AND user_id = ?", id, memberUserID).Delete(&HouseholdMember{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("member not found in household")
	}

	// Notify the household members about the removed member
	var household Household
	if err := s.db.First(&household, id).Error; err != nil {
		return fmt.Errorf("failed to get household: %w", err)
	}

	s.notify(context.Background(), notification.NotificationTypeMemberRemoved,
		"Household member removed",
		fmt.Sprintf("A member was removed from household %q", household.Name),
		household.ID, 0, 0, ownerID)

	return nil
}

// notify delivers a notification to all household members.
func (s *Service) notify(ctx context.Context, nt notification.NotificationType, title, body string, householdID, listID, itemID, actorID uint) {
	if s.notification == nil {
		return
	}

	if err := s.notification.NotifyHousehold(ctx, notification.BuildNotification(nt, title, body, householdID, listID, itemID, actorID)); err != nil {
		slog.Error("household notification error", "error", err)
	}
}

// checkOwner verifies that the given user is the owner of the household.
func (s *Service) checkOwner(householdID, userID uint) error {
	var household Household
	if err := s.db.First(&household, householdID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("household not found")
		}
		return fmt.Errorf("failed to get household: %w", err)
	}

	if household.OwnerID != userID {
		return errors.New("only the household owner can perform this action")
	}

	return nil
}