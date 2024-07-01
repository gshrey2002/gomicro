package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/unusualcodeorg/gomicro/auth-service/api/auth/dto"
	"github.com/unusualcodeorg/gomicro/auth-service/api/auth/message"
	"github.com/unusualcodeorg/gomicro/auth-service/api/user"
	"github.com/unusualcodeorg/gomicro/auth-service/common"
	"github.com/unusualcodeorg/goserve/arch/micro"
	"github.com/unusualcodeorg/goserve/arch/network"
	"github.com/unusualcodeorg/goserve/utils"
)

type controller struct {
	micro.BaseController
	common.ContextPayload
	service     Service
	userService user.Service
}

func NewController(
	authProvider network.AuthenticationProvider,
	authorizeProvider network.AuthorizationProvider,
	service Service,
	userService user.Service,
) micro.Controller {
	return &controller{
		BaseController: micro.NewBaseController("/", authProvider, authorizeProvider),
		ContextPayload: common.NewContextPayload(),
		service:        service,
		userService:    userService,
	}
}

func (c *controller) MountNats(group micro.NatsGroup) {
	group.AddEndpoint("authentication", micro.NatsHandlerFunc(c.authenticationHandler))
	group.AddEndpoint("authorization", micro.NatsHandlerFunc(c.authorizationHandler))
}

func (c *controller) authenticationHandler(req micro.NatsRequest) {
	text, err := micro.ParseMsg[message.Text](req.Data())
	if err != nil {
		c.SendNats(req).Error(err)
		return
	}

	user, _, err := c.service.Authenticate(text.Value)
	if err != nil {
		c.SendNats(req).Error(err)
		return
	}

	c.SendNats(req).Message(message.NewUser(user))
}

func (c *controller) authorizationHandler(req micro.NatsRequest) {
	userRole, err := micro.ParseMsg[message.UserRole](req.Data())
	if err != nil {
		c.SendNats(req).Error(err)
		return
	}

	user, err := c.userService.FindUserById(userRole.User.ID)
	if err != nil {
		c.SendNats(req).Error(err)
		return
	}

	err = c.service.Authorize(user, userRole.Roles...)
	if err != nil {
		c.SendNats(req).Error(err)
		return
	}

	c.SendNats(req).Message(message.NewUser(user))
}

func (c *controller) MountRoutes(group *gin.RouterGroup) {
	group.GET("/ping", c.pingHandler)
	group.POST("/signup/basic", c.signUpBasicHandler)
	group.POST("/signin/basic", c.signInBasicHandler)
	group.POST("/token/refresh", c.tokenRefreshHandler)
	group.DELETE("/signout", c.Authentication(), c.signOutBasic)
}

func (c *controller) pingHandler(ctx *gin.Context) {
	c.Send(ctx).SuccessMsgResponse("success")
}

func (c *controller) signUpBasicHandler(ctx *gin.Context) {
	body, err := network.ReqBody(ctx, dto.EmptySignUpBasic())
	if err != nil {
		c.Send(ctx).BadRequestError(err.Error(), err)
		return
	}

	data, err := c.service.SignUpBasic(body)
	if err != nil {
		c.Send(ctx).MixedError(err)
		return
	}

	c.Send(ctx).SuccessDataResponse("success", data)
}

func (c *controller) signInBasicHandler(ctx *gin.Context) {
	body, err := network.ReqBody(ctx, dto.EmptySignInBasic())
	if err != nil {
		c.Send(ctx).BadRequestError(err.Error(), err)
		return
	}

	dto, err := c.service.SignInBasic(body)
	if err != nil {
		c.Send(ctx).MixedError(err)
		return
	}

	c.Send(ctx).SuccessDataResponse("success", dto)
}

func (c *controller) signOutBasic(ctx *gin.Context) {
	keystore := c.MustGetKeystore(ctx)

	err := c.service.SignOut(keystore)
	if err != nil {
		c.Send(ctx).InternalServerError("something went wrong", err)
		return
	}

	c.Send(ctx).SuccessMsgResponse("signout success")
}

func (c *controller) tokenRefreshHandler(ctx *gin.Context) {
	body, err := network.ReqBody(ctx, dto.EmptyTokenRefresh())
	if err != nil {
		c.Send(ctx).BadRequestError(err.Error(), err)
		return
	}

	authHeader := ctx.GetHeader(network.AuthorizationHeader)
	accessToken := utils.ExtractBearerToken(authHeader)

	dto, err := c.service.RenewToken(body, accessToken)
	if err != nil {
		c.Send(ctx).MixedError(err)
		return
	}

	c.Send(ctx).SuccessDataResponse("success", dto)
}
