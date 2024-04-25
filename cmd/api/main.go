package main

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
)

type ValidationError struct {
	HasError bool        `json:"has_error"`
	Field    string      `json:"field"`
	Tag      string      `json:"tag"`
	Value    interface{} `json:"value"`
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(data interface{}) []ValidationError {
	var validationErrors []ValidationError

	errs := cv.validator.Struct(data)
	if errs != nil {
		for _, err := range errs.(validator.ValidationErrors) {
			var ve ValidationError
			ve.Value = err.Value()
			ve.Field = err.Field()
			ve.Tag = err.Tag()
			ve.HasError = true

			validationErrors = append(validationErrors, ve)
		}
	}
	return validationErrors
}

func main() {
	customValidator := &CustomValidator{validator: validator.New()}

	if err := customValidator.validator.RegisterValidation("oldAge",
		func(fl validator.FieldLevel) bool {
			return fl.Field().Int() < 40
		}); err != nil {
		panic("wow")
	}

	app := fiber.New()

	app.Use(recover.New())

	app.Use(requestid.New())
	app.Use(func(ctx *fiber.Ctx) error {
		fmt.Println("You have called " + string(ctx.Request().RequestURI()))
		return ctx.Next()
	})

	app.Use("/orders/code/:code", func(ctx *fiber.Ctx) error {
		correlationID := ctx.Get("x-correlation-id")

		if correlationID == "" {
			return ctx.Status(fiber.StatusBadRequest).JSON("Correlation ID is required")
		}

		_, err := uuid.Parse(correlationID)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON("Correlation ID is not guid")
		}

		ctx.Locals("correlationID", correlationID)
		return ctx.Next()
	})

	app.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.SendString("Hello, World!")
	})
	app.Get("/ping", func(ctx *fiber.Ctx) error {
		return ctx.SendString("pong")
	})

	app.Get("/orders/code/:code", func(ctx *fiber.Ctx) error {
		fmt.Printf("Your correlation id is %v\n", ctx.Locals("correlationID"))
		return ctx.SendString(fmt.Sprintf("Order code: %v", ctx.Params("code")))
	})

	app.Post("/orders/", func(ctx *fiber.Ctx) error {
		var req CreateOrderRequest
		if err := ctx.BodyParser(&req); err != nil {
			return err
		}

		if errs := customValidator.Validate(req); len(errs) != 0 && errs[0].HasError {
			errorMessages := make([]string, 0)

			for _, err := range errs {
				errMsg := fmt.Sprintf(
					"%s field has failed. Validation is: %s",
					err.Field,
					err.Tag,
				)
				errorMessages = append(errorMessages, errMsg)
			}
			return ctx.Status(fiber.StatusBadRequest).JSON(strings.Join(
				errorMessages,
				" and that ",
			))
		}

		return ctx.Status(fiber.StatusCreated).JSON("Order created successfully")
	})
	app.Listen(":3000")
}

type CreateOrderRequest struct {
	ShipmentNumber string `json:"shipment_number" validate:"required"`
	Age            int    `json:"age" validate:"required,oldAge"`
}
