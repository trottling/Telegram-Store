package dto

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
)

// Проверяем теги тем же валидатором, который вызывает ShouldBindJSON: ошибка в
// имени тега не ловится компилятором и проявилась бы отказом на валидных
// запросах — например, невозможностью войти в панель.

func TestExchangeRequestBinding(t *testing.T) {
	tests := []struct {
		name     string
		initData string
		wantErr  bool
	}{
		{"непустая initData", "user=...&hash=...", false},
		{"пусто", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := binding.Validator.ValidateStruct(&ExchangeRequest{InitData: tt.initData})
			if (err != nil) != tt.wantErr {
				t.Errorf("initData %q: ошибка = %v, ожидалась ошибка: %v", tt.initData, err, tt.wantErr)
			}
		})
	}
}

func TestProductRequestBinding(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateProductRequest
		wantErr bool
	}{
		{"валидный", CreateProductRequest{Name: "ключ", Price: 100}, false},
		{"без имени", CreateProductRequest{Price: 100}, true},
		{"нулевая цена", CreateProductRequest{Name: "ключ"}, true},
		{"отрицательная цена", CreateProductRequest{Name: "ключ", Price: -1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := binding.Validator.ValidateStruct(&tt.req); (err != nil) != tt.wantErr {
				t.Errorf("ошибка = %v, ожидалась ошибка: %v", err, tt.wantErr)
			}
		})
	}
}

// TestUpdateProductRequestAllowsDeactivation — ловушка с bool: повесив на
// IsActive required, мы сделали бы невозможным именно деактивацию товара,
// потому что false для валидатора неотличим от «поля нет».
func TestUpdateProductRequestAllowsDeactivation(t *testing.T) {
	req := &UpdateProductRequest{Name: "ключ", Price: 100, IsActive: false}
	if err := binding.Validator.ValidateStruct(req); err != nil {
		t.Errorf("деактивация товара отвергнута: %v", err)
	}
}

// TestUpdateBalanceRequestBinding — required на числе означает «не ноль», что
// совпадает с проверкой AdminSrv.AddBalance.
func TestUpdateBalanceRequestBinding(t *testing.T) {
	if err := binding.Validator.ValidateStruct(&UpdateBalanceRequest{Amount: 100}); err != nil {
		t.Errorf("начисление отвергнуто: %v", err)
	}
	if err := binding.Validator.ValidateStruct(&UpdateBalanceRequest{Amount: -100}); err != nil {
		t.Errorf("списание отвергнуто: %v", err)
	}
	if err := binding.Validator.ValidateStruct(&UpdateBalanceRequest{Amount: 0}); err == nil {
		t.Error("нулевая корректировка принята, ожидался отказ")
	}
}

func TestAddItemsRequestBinding(t *testing.T) {
	if err := binding.Validator.ValidateStruct(&AddItemsRequest{Contents: []string{"key-1"}}); err != nil {
		t.Errorf("непустой список отвергнут: %v", err)
	}
	if err := binding.Validator.ValidateStruct(&AddItemsRequest{Contents: []string{}}); err == nil {
		t.Error("пустой список принят, ожидался отказ")
	}
}

func TestCategoryAndSettingsRequestBinding(t *testing.T) {
	if err := binding.Validator.ValidateStruct(&CreateCategoryRequest{Name: "Игры"}); err != nil {
		t.Errorf("категория с именем отвергнута: %v", err)
	}
	if err := binding.Validator.ValidateStruct(&CreateCategoryRequest{}); err == nil {
		t.Error("категория без имени принята, ожидался отказ")
	}
	if err := binding.Validator.ValidateStruct(&UpdateSettingsRequest{SupportUsername: "support"}); err != nil {
		t.Errorf("настройки с support_username отвергнуты: %v", err)
	}
	if err := binding.Validator.ValidateStruct(&UpdateSettingsRequest{}); err == nil {
		t.Error("настройки без support_username приняты, ожидался отказ")
	}
}
