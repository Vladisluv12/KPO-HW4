// services/payment_processor.go
package services

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

type PaymentProcessor struct {
	paymentService PaymentService
	messageService MessageService
}

func NewPaymentProcessor(paymentService PaymentService, messageService MessageService) *PaymentProcessor {
	return &PaymentProcessor{
		paymentService: paymentService,
		messageService: messageService,
	}
}

func (p *PaymentProcessor) ProcessMessages(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processBatch(ctx)
		}
	}
}

func (p *PaymentProcessor) processBatch(ctx context.Context) {
	// Получаем непрочитанные сообщения из inbox
	messages, err := p.messageService.GetUnprocessedMessages(ctx, "payments.payment_requests", 10)
	if err != nil {
		log.Printf("Error getting unprocessed messages: %v", err)
		return
	}

	for _, msg := range messages {
		var request PaymentRequest
		if err := json.Unmarshal(msg.Payload, &request); err != nil {
			log.Printf("Error unmarshaling payment request: %v", err)
			p.messageService.MarkMessageProcessed(ctx, msg.MessageID)
			continue
		}

		// Обрабатываем платеж
		result, err := p.paymentService.ProcessPayment(ctx, request)
		if err != nil {
			log.Printf("Error processing payment: %v", err)
			// Можно добавить retry логику здесь
		}

		log.Printf("Payment processed: OrderID=%s, Status=%s", result.OrderID, result.Status)

		// Помечаем сообщение как обработанное
		if err := p.messageService.MarkMessageProcessed(ctx, msg.MessageID); err != nil {
			log.Printf("Error marking message as processed: %v", err)
		}
	}
}
