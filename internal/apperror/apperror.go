package apperror

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code    string
	Message string
}

// Error implements the error interface for AppError
func (e AppError) Error() string {
	return e.Message
}

// Code extracts the error Code from an error, returning the AppError Code if it's an AppError, otherwise returns INTERNAL_ERROR
func Code(err error) string {
	if isAppError(err) {
		return err.(AppError).Code
	}
	return ErrInternal.Code
}
func NewError(originalErr error, code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Message extracts the error Message from an error, returning the AppError Message if it's an AppError, otherwise returns a generic internal error Message
func Message(err error) string {
	if isAppError(err) {
		return err.(AppError).Message
	}
	return ErrInternal.Message
}

// isAppError checks if an error is an AppError (safe to expose to frontend)
func isAppError(err error) bool {
	var appError AppError
	ok := errors.As(err, &appError)
	return ok
}

// isErrorType checks if err matches any of the provided target errors
func isErrorType(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// StatusFromError maps custom errors to HTTP status codes
func StatusFromError(err error) int {
	switch {
	// 400 Bad Request
	case isErrorType(err, ErrBadRequest, ErrInvalidID, ErrInvalidMembershipData, ErrInvalidOTP, ErrOTPExpired,
		ErrInvalidGender, ErrInvalidDateFormat, ErrAgeTooYoung, ErrInvalidBirthDate, ErrInvalidProvince, ErrTooManyInterests, ErrInvalidInterest, ErrInsufficientStock,
		ErrOrderCannotBeCanceled, ErrOrderCannotBeReturned, ErrInvalidOrderStatus, ErrInvalidOrderStatusTransition, ErrInvalidPaymentMethod, ErrMultipleSellersNotAllowed,
		ErrCategoryHasChildren, ErrCategoryHasProducts, ErrInvalidParentCategory):
		return http.StatusBadRequest
	// 401 Unauthorized
	case isErrorType(err, ErrInvalidCredentials, ErrInvalidToken, ErrInvalidClaims, ErrInvalidIssuer, ErrInvalidAudience, ErrTokenInvalidated, ErrMissingAuthHeader,
		ErrInvalidAuthHeader, ErrMissingToken, ErrNotAuthenticated, ErrSellerAccessRequired):
		return http.StatusUnauthorized
	// 403 Forbidden
	case isErrorType(err, ErrForbidden, ErrUserInactive, ErrUserNotMember, ErrEmailNotVerified, ErrAdminAccessRequired,
		ErrProvinceHasWards, ErrProductNotAvailable, ErrVariantNotFound, ErrVariantRequired):
		return http.StatusForbidden
	// 404 Not Found
	case isErrorType(err, ErrUserNotFound, ErrCommunityNotFound, ErrCommunityDeleted, ErrMembershipNotFound, ErrProductNotFound, ErrCartNotFound, ErrCartItemNotFound,
		ErrPostNotFound, ErrVoteNotFound, ErrDraftNotFound, ErrEmailNotRegistered, ErrProvinceNotFound, ErrWardNotFound, ErrOrderNotFound, ErrCategoryNotFound):
		return http.StatusNotFound
	// 409 Conflict
	case isErrorType(err, ErrUsernameExists, ErrEmailExists, ErrCommunityNameExists,
		ErrAlreadyMember, ErrEmailAlreadyVerified, ErrLoginMethodMismatch, ErrPollVoted, ErrPollCannotEdit, ErrAlreadyReported, ErrCategoryNameExists):
		return http.StatusConflict
	// 500 Internal Server Error
	case isErrorType(err, ErrInternal, ErrNoFieldsToUpdate, ErrMembershipCreateFailed, ErrMembershipDeleteFailed):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

var (
	// Auth-related
	ErrInvalidCredentials   = AppError{Code: "INVALID_CREDENTIALS", Message: "Email or password is incorrect"}
	ErrInvalidToken         = AppError{Code: "INVALID_TOKEN", Message: "Token is invalid or expired"}
	ErrInvalidClaims        = AppError{Code: "INVALID_CLAIMS", Message: "Invalid token claims"}
	ErrInvalidIssuer        = AppError{Code: "INVALID_ISSUER", Message: "Invalid token issuer"}
	ErrInvalidAudience      = AppError{Code: "INVALID_AUDIENCE", Message: "Invalid token audience"}
	ErrTokenInvalidated     = AppError{Code: "TOKEN_INVALIDATED", Message: "Token has been invalidated"}
	ErrMissingAuthHeader    = AppError{Code: "MISSING_AUTH_HEADER", Message: "Missing Authorization header"}
	ErrInvalidAuthHeader    = AppError{Code: "INVALID_AUTH_HEADER", Message: "Invalid Authorization header format"}
	ErrMissingToken         = AppError{Code: "MISSING_TOKEN", Message: "Missing authentication token"}
	ErrNotAuthenticated     = AppError{Code: "NOT_AUTHENTICATED", Message: "Not authenticated"}
	ErrInvalidAuthContext   = AppError{Code: "INVALID_AUTH_CONTEXT", Message: "Invalid authentication context"}
	ErrAdminAccessRequired  = AppError{Code: "ADMIN_ACCESS_REQUIRED", Message: "Admin access required"}
	ErrSellerAccessRequired = AppError{Code: "SELLER_ACCESS_REQUIRED", Message: "Seller access required"}
	ErrForbidden            = AppError{Code: "FORBIDDEN", Message: "You do not have permission to perform this action"}
	ErrBadRequest           = AppError{Code: "BAD_REQUEST", Message: "Bad request"}
	ErrEmailNotVerified     = AppError{Code: "EMAIL_NOT_VERIFIED", Message: "Email not verified"}
	ErrEmailAlreadyVerified = AppError{Code: "EMAIL_ALREADY_VERIFIED", Message: "Email already verified"}
	ErrInvalidOTP           = AppError{Code: "INVALID_OTP", Message: "Invalid OTP"}
	ErrOTPExpired           = AppError{Code: "OTP_EXPIRED", Message: "OTP has expired"}
	ErrLoginMethodMismatch  = AppError{Code: "LOGIN_METHOD_MISMATCH", Message: "This email has been registered using a different login method. Please use the original login method."}
	ErrEmailNotRegistered   = AppError{Code: "EMAIL_NOT_REGISTERED", Message: "Email not registered"}
	// Generic
	ErrInternal          = AppError{Code: "INTERNAL_ERROR", Message: "Internal server error"}
	ErrNoFieldsToUpdate  = AppError{Code: "NO_FIELDS_TO_UPDATE", Message: "No fields to update"}
	ErrInvalidID         = AppError{Code: "INVALID_ID", Message: "Invalid ID format"}
	ErrPaginationInvalid = AppError{Code: "PAGINATION_INVALID", Message: "Invalid page number or page size. Page size must be less than 500."}

	// User-related
	ErrUserNotFound   = AppError{Code: "USER_NOT_FOUND", Message: "User not found"}
	ErrUsernameExists = AppError{Code: "USERNAME_EXISTS", Message: "Username already exists"}
	ErrEmailExists    = AppError{Code: "EMAIL_EXISTS", Message: "Email already exists"}
	ErrUserInactive   = AppError{Code: "USER_INACTIVE", Message: "User account has been deactivated"}

	// Profile validation
	ErrInvalidGender     = AppError{Code: "INVALID_GENDER", Message: "Invalid gender value"}
	ErrInvalidDateFormat = AppError{Code: "INVALID_DATE_FORMAT", Message: "Invalid date format, use YYYY-MM-DD"}
	ErrAgeTooYoung       = AppError{Code: "AGE_TOO_YOUNG", Message: "Must be at least 13 years old"}
	ErrInvalidBirthDate  = AppError{Code: "INVALID_BIRTH_DATE", Message: "Invalid birth date"}
	ErrInvalidProvince   = AppError{Code: "INVALID_PROVINCE", Message: "Invalid province"}
	ErrTooManyInterests  = AppError{Code: "TOO_MANY_INTERESTS", Message: "Maximum of 10 interests"}
	ErrInvalidInterest   = AppError{Code: "INVALID_INTEREST", Message: "Invalid interest"}
	// Community-related
	ErrCommunityNotFound         = AppError{Code: "COMMUNITY_NOT_FOUND", Message: "Community not found"}
	ErrCommunityNameExists       = AppError{Code: "COMMUNITY_NAME_EXISTS", Message: "Community name already exists"}
	ErrCommunityDeleted          = AppError{Code: "COMMUNITY_DELETED", Message: "Community has been deleted"}
	ErrModeratorAlreadyExists    = AppError{Code: "MODERATOR_ALREADY_EXISTS", Message: "User is already a moderator of the community."}
	ErrCannotRemoveModerator     = AppError{Code: "CANNOT_REMOVE_MODERATOR", Message: "Cannot remove this moderator."}
	ErrCannotRemoveCreator       = AppError{Code: "CANNOT_REMOVE_CREATOR", Message: "Cannot remove the community creator from the list of moderators."}
	ErrUserIsBannedFromCommunity = AppError{Code: "BANNED_COMMUNITY", Message: "User has been banned from the community."}

	// Membership-related
	ErrUserNotMember          = AppError{Code: "USER_NOT_MEMBER", Message: "You are not a member of this community"}
	ErrMembershipNotFound     = AppError{Code: "MEMBERSHIP_NOT_FOUND", Message: "Member not found"}
	ErrAlreadyMember          = AppError{Code: "ALREADY_MEMBER", Message: "You are already a member of this community"}
	ErrMembershipCreateFailed = AppError{Code: "MEMBERSHIP_CREATE_FAILED", Message: "Failed to create member"}
	ErrMembershipDeleteFailed = AppError{Code: "MEMBERSHIP_DELETE_FAILED", Message: "Failed to delete member"}
	ErrInvalidMembershipData  = AppError{Code: "INVALID_MEMBERSHIP_DATA", Message: "Invalid membership data"}

	// Post-related
	ErrPostNotFound    = AppError{Code: "POST_NOT_FOUND", Message: "Post not found"}
	ErrVoteNotFound    = AppError{Code: "VOTE_NOT_FOUND", Message: "Vote not found"}
	ErrPollVoted       = AppError{Code: "POLL_ALREADY_VOTED", Message: "You have already voted for this option"}
	ErrPollCannotEdit  = AppError{Code: "POLL_CANNOT_EDIT", Message: "Cannot edit poll after voting has occurred"}
	ErrAlreadyReported = AppError{Code: "ALREADY_REPORTED", Message: "You have already reported this content"}
	ErrDraftNotFound   = AppError{Code: "DRAFT_NOT_FOUND", Message: "Draft not found"}

	// Product-related
	ErrProductNotFound     = AppError{Code: "PRODUCT_NOT_FOUND", Message: "Product not found"}
	ErrProductNotAvailable = AppError{Code: "PRODUCT_NOT_AVAILABLE", Message: "Product not available"}
	ErrVariantNotFound     = AppError{Code: "VARIANT_NOT_FOUND", Message: "Product variant not found"}
	ErrVariantRequired     = AppError{Code: "VARIANT_REQUIRED", Message: "Variant selection is required for this product"}
	ErrInsufficientStock   = AppError{Code: "INSUFFICIENT_STOCK", Message: "Insufficient stock available"}

	// Cart-related
	ErrCartNotFound     = AppError{Code: "CART_NOT_FOUND", Message: "Cart not found"}
	ErrCartItemNotFound = AppError{Code: "CART_ITEM_NOT_FOUND", Message: "Cart item not found"}
	ErrProvinceNotFound = AppError{Code: "PROVINCE_NOT_FOUND", Message: "Province not found"}
	ErrProvinceHasWards = AppError{Code: "PROVINCE_HAS_WARDS", Message: "Cannot delete province with wards"}
	ErrWardNotFound     = AppError{Code: "WARD_NOT_FOUND", Message: "Ward not found"}

	ErrOrderNotFound                = AppError{Code: "ORDER_NOT_FOUND", Message: "Order not found"}
	ErrOrderCannotBeCanceled        = AppError{Code: "ORDER_CANNOT_BE_CANCELED", Message: "Order cannot be canceled in the current status"}
	ErrOrderCannotBeReturned        = AppError{Code: "ORDER_CANNOT_BE_RETURNED", Message: "Order cannot be returned in the current status"}
	ErrInvalidOrderStatus           = AppError{Code: "INVALID_ORDER_STATUS", Message: "Invalid order status"}
	ErrInvalidOrderStatusTransition = AppError{Code: "INVALID_ORDER_STATUS_TRANSITION", Message: "Invalid order status transition"}
	ErrInvalidPaymentMethod         = AppError{Code: "INVALID_PAYMENT_METHOD", Message: "Invalid payment method"}
	ErrMultipleSellersNotAllowed    = AppError{Code: "MULTIPLE_SELLERS_NOT_ALLOWED", Message: "An order can only be purchased from one seller"}

	// Category-related
	ErrCategoryNotFound      = AppError{Code: "CATEGORY_NOT_FOUND", Message: "Category not found"}
	ErrCategoryNameExists    = AppError{Code: "CATEGORY_NAME_EXISTS", Message: "Category name already exists"}
	ErrCategoryHasChildren   = AppError{Code: "CATEGORY_HAS_CHILDREN", Message: "Cannot delete category with child categories"}
	ErrCategoryHasProducts   = AppError{Code: "CATEGORY_HAS_PRODUCTS", Message: "Cannot delete category with products"}
	ErrInvalidParentCategory = AppError{Code: "INVALID_PARENT_CATEGORY", Message: "Invalid parent category"}
)
