package tg

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func createRootMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("RootMenu", "show_root_menu"),
		),
	)
}

func createAdminMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("AdminMenu", "show_admin_menu"),
		),
	)
}

func createUserMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Меню", "show_user_menu"),
		),
	)
}

func createRootCommandsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Добавить решение", "create_code"),
			tgbotapi.NewInlineKeyboardButtonData("Получить решение", "get_code"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалить решение", "del_code"),
			tgbotapi.NewInlineKeyboardButtonData("Обновить решение", "up_code"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Добавить админа", "new_admin"),
			tgbotapi.NewInlineKeyboardButtonData("Удалить админа", "del_admin"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Получить список моих запросов", "get_my_request"),
		),
	)
}

func createAdminCommandsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Добавить решение", "create_code"),
			tgbotapi.NewInlineKeyboardButtonData("Получить решение", "get_code"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалить решение", "del_code"),
			tgbotapi.NewInlineKeyboardButtonData("Обновить решение", "up_code"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Получить список моих запросов", "get_my_request"),
		),
	)
}

func createUserCommandsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Получить решение", "get_code"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Получить список моих запросов", "get_my_request"),
		),
	)
}

func createCheckPasswordKeybord() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Пароль верный", "yes"),
			tgbotapi.NewInlineKeyboardButtonData("Пароль неверный", "no"),
		),
	)
}
