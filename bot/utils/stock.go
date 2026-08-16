package utils

// StockIndicator — эмодзи-светофор по остатку товара, просто подсказка.
func StockIndicator(count int) string {
	switch {
	case count <= 3:
		return "🔴"
	case count <= 5:
		return "🟠"
	case count <= 7:
		return "🟡"
	default:
		return "🟢"
	}
}
