package auth

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/unusualcodeorg/gomicro/auth-service/api/auth/dto"
	"github.com/unusualcodeorg/gomicro/auth-service/api/user"
	"github.com/unusualcodeorg/goserve/arch/network"
)

func TestAuthController_SignupBadRequest(t *testing.T) {
	mockAuthProvider := new(network.MockAuthenticationProvider)
	mockAuthProvider.On("Middleware").Return(gin.HandlerFunc(func(ctx *gin.Context) {
		ctx.Next()
	}))

	mockAuthzProvider := new(network.MockAuthorizationProvider)
	mockAuthzProvider.On("Middleware", "ROLE").Return(gin.HandlerFunc(func(ctx *gin.Context) {
		ctx.Next()
	}))

	authService := new(MockService)
	userService := new(user.MockService)

	c := NewController(mockAuthProvider, mockAuthzProvider, authService, userService)

	rr := network.MockTestController(t, "POST", "/signup/basic", "{}", c.(network.Controller))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), `"message":"email is required, password is required, name is required"`)
}

func TestAuthController_SignupSuccess(t *testing.T) {
	mockAuthProvider := new(network.MockAuthenticationProvider)
	mockAuthProvider.On("Middleware").Return(gin.HandlerFunc(func(ctx *gin.Context) {
		ctx.Next()
	}))

	mockAuthzProvider := new(network.MockAuthorizationProvider)
	mockAuthzProvider.On("Middleware", "ROLE").Return(gin.HandlerFunc(func(ctx *gin.Context) {
		ctx.Next()
	}))

	body := `{"email":"test@abc.com","password":"123456","name":"test name"}`

	singUpDto := &dto.SignUpBasic{
		Email:    "test@abc.com",
		Password: "123456",
		Name:     "test name",
	}

	authService := new(MockService)
	userService := new(user.MockService)

	authService.On("SignUpBasic", singUpDto).Return(&dto.UserAuth{}, nil)

	c := NewController(mockAuthProvider, mockAuthzProvider, authService, userService)

	rr := network.MockTestController(t, "POST", "/signup/basic", body, c.(network.Controller))
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"message":"success"`)
}
