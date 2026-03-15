package tg

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"
	"unicode"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func showMainMenu(bot *tgbotapi.BotAPI, chatID int64, from *tgbotapi.User) {
	log.Printf("sMM: started showMainMenu for user %s;", from.UserName)
	timer := time.NewTimer(1 * time.Minute)
	tiker := time.NewTicker(250 * time.Millisecond)
	done := make(chan struct{})

	go func() {
		defer close(done)
		tik := 0
		defer func() {
			if r := recover(); r != nil {
				log.Printf("sMM: intercepted panic: %v", r)
				sendErrorMessage(bot, chatID, "Что-то пошло не так, попробуйте снова", "", "")
				return
			}
		}()

		ctxGUS, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, status_in_db, password_in_db, err := pg_us_db.GetUserStatus(ctxGUS, from.ID)

		var status string
		if from.ID == conf.ROOT_ID {
			status = "root"
		} else {
			if status_in_db == "admin" {
				status = "admin"
			} else {
				status = "user"
			}
		}

		if err != nil && err != sql.ErrNoRows {
			log.Printf("sMM: error getting user status: %v", err)
			status = ""
			sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 1", "", "")
			return
		} else if err == sql.ErrNoRows {
			switch status {
			case "root":
				status = ""
				hashPassword := RootAndAdminHandler(bot, chatID, from, false)
				if hashPassword == "" {
					log.Printf("sMM: root password setup failed")
					return
				}

				ctxIU, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				log.Printf("sMM: inserting root user ID: %d, Login: %s", from.ID, from.UserName)
				if err := pg_us_db.InsertUser(ctxIU, from.ID, chatID, "root", hashPassword, from.UserName, "", 0); err != nil {
					log.Printf("sMM: error inserting root user: %v", err)
					sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 2", "", "")
					return
				}
				sendSuccessMessage(bot, chatID, "Ваш пароль успешно сохранён.\nНажмите /docx чтобы ознакомиться с инструкцией в формате файла (.docx)", "", "")
				return
			case "user":
				status = ""
				ctxIU, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				log.Printf("sMM: inserting user ID: %d, Login: %s", from.ID, from.UserName)
				if err := pg_us_db.InsertUser(ctxIU, from.ID, chatID, "user", "", from.UserName, "", 0); err != nil {
					log.Printf("sMM: error inserting user: %v", err)
					sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 3", "", "")
					return
				}
				sendSuccessMessage(bot, chatID, fmt.Sprintf(
					"Привет %s.\nНажми /docx чтобы ознакомиться с инструкцией в формате файла (.docx)",
					strings.TrimSpace(from.LastName+" "+from.FirstName),
				), "", "",
				)
				return
			default:
				status = ""
				log.Printf("sMM: unknown status for user ID: %d", from.ID)
				sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 4", "", "")
				return
			}
		} else if err == nil {
			switch status {
			case "root":
				status = ""
				if password_in_db == "" {
					hashPassword := RootAndAdminHandler(bot, chatID, from, false)
					if hashPassword == "" {
						log.Printf("sMM: root password setup failed")
						return
					}

					ctxUU, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := pg_us_db.UpdateUserStatus(ctxUU, from.ID, "root", hashPassword); err != nil {
						log.Printf("sMM: error updating root user status: %v", err)
						sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 5", "", "")
						return
					}
					sendSuccessMessage(bot, chatID, "Ваш пароль успешно сохранён.", "root", "")
					return
				}
				sendSuccessMessage(bot, chatID, "Введите старый пароль", "root", "")
				setUserState(chatID, map[string]string{"re_password": ""})
				for range tiker.C {
					tik++
					if tik >= 240 {
						tik = 0
						return
					}
					ind, ok := getUserState(chatID)
					if ok && ind["re_password"] == "true" {
						hashPassword := RootAndAdminHandler(bot, chatID, from, true)
						if hashPassword == "" {
							log.Printf("sMM: root password setup failed")
							return
						}

						ctxUU, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						if err := pg_us_db.UpdateUserStatus(ctxUU, from.ID, "root", hashPassword); err != nil {
							log.Printf("sMM: error updating root user status: %v", err)
							sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 5", "", "")
							return
						}
						sendSuccessMessage(bot, chatID, "Ваш пароль успешно сохранён.", "root", "")
						return
					}
					if ok && ind["re_password"] == "false" {
						setUserState(chatID, map[string]string{"re_password": "waiting for next input"})
						sendErrorMessage(bot, chatID, "Пароль не верный, попробуйте снова", "root", "")
					}
					if !ok {
						deleteUserState(chatID)
						sendErrorMessage(bot, chatID, "error when accessing map", "", "")
						return
					}
					runtime.Gosched()
				}
			case "admin":
				status = ""
				if password_in_db == "" {
					hashPassword := RootAndAdminHandler(bot, chatID, from, true)
					if hashPassword == "" {
						log.Printf("sMM: root password setup failed")
						return
					}

					ctxUU, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := pg_us_db.UpdateUserStatus(ctxUU, from.ID, "admin", hashPassword); err != nil {
						log.Printf("sMM: error updating root user status: %v", err)
						sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 5", "", "")
						return
					}
					sendSuccessMessage(bot, chatID, "Ваш пароль успешно сохранён.", "admin", "")
					return
				}
				sendSuccessMessage(bot, chatID, "Введите старый пароль", "admin", "")
				setUserState(chatID, map[string]string{"re_password": ""})
				for range tiker.C {
					tik++
					if tik >= 240 {
						tik = 0
						return
					}
					ind, ok := getUserState(chatID)
					if ok && ind["re_password"] == "true" {
						hashPassword := RootAndAdminHandler(bot, chatID, from, true)
						if hashPassword == "" {
							log.Printf("sMM: admin password setup failed")
							return
						}

						ctxUU, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						if err := pg_us_db.UpdateUserStatus(ctxUU, from.ID, "admin", hashPassword); err != nil {
							log.Printf("sMM: error updating admin user status: %v", err)
							sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 6", "", "")
							return
						}
						sendSuccessMessage(bot, chatID, "Ваш пароль успешно сохранён.", "admin", "")
						return
					}
					if ok && ind["re_password"] == "false" {
						setUserState(chatID, map[string]string{"re_password": "waiting for next input"})
						sendErrorMessage(bot, chatID, "Пароль не верный, попробуйте снова", "admin", "")
					}
					if !ok {
						deleteUserState(chatID)
						sendErrorMessage(bot, chatID, "error when accessing map", "", "")
						return
					}
					runtime.Gosched()
				}
			case "user":
				status = ""
				sendSuccessMessage(bot, chatID, "Нажми /docx чтобы ознакомиться с инструкцией в формате файла (.docx)", "", "")
				return
			default:
				status = ""
				log.Printf("sMM: unknown status for user ID: %d", from.ID)
				sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 7", "", "")
				return
			}
		}
	}()

	select {
	case <-timer.C:
		log.Printf("sMM: session timed out for user ID: %d", chatID)
		deleteUserState(chatID)
		sendErrorMessage(bot, chatID, "Ваша сессия ввода пароля истекла, чтобы продолжить нажмите /start повторно", "", "")
		return
	case <-done:
		deleteUserState(chatID)
		timer.Stop()
		log.Printf("sMM: completed for user ID: %d", chatID)
		return
	}
}

func handlePassword(bot *tgbotapi.BotAPI, chatID int64, message string, messageID int) {
	log.Printf("hP: started handlePassword")
	timer := time.NewTimer(1 * time.Minute)
	tiker := time.NewTicker(250 * time.Millisecond)
	done := make(chan struct{})

	go func() {
		defer close(done)
		tik := 0
		runa, ok := isValid(message)
		log.Printf("hP: password validation result: %v", ok)
		if !ok {
			if string(runa) == "" {
				sendErrorMessage(bot, chatID, "Пароль не может быть пустым", "", "")
				log.Printf("hP: password is empty")
				return
			}
			sendErrorMessage(bot, chatID, fmt.Sprintf("Ваш пароль содержит запрещённый символ: '%s'.\nВведите пароль корректно!", string(runa)), "", "")
			delUser := tgbotapi.NewDeleteMessage(chatID, messageID)
			bot.Request(delUser)
			log.Printf("hP: invalid character in password: %c", runa)
			setUserState(chatID, map[string]string{"password": ""})
			return
		}

		hash := sha256.Sum256([]byte(message))
		hashString := hex.EncodeToString(hash[:])

		mdV2 := escapeMarkdownV2(message)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ваш пароль получен ||%s||\\.\n После нажатия любой кнопки сообщения будут удалены\\.\nПароль верный?", mdV2))
		msg.ReplyMarkup = createCheckPasswordKeybord()
		msg.ParseMode = "MarkdownV2"
		sendMsg, err := bot.Send(msg)
		if err != nil {
			log.Printf("error whem sending message: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error: повторите попытку позже.", "", "")
			return
		}

		for range tiker.C {
			tik++
			if tik >= 240 {
				tik = 0
				delUser := tgbotapi.NewDeleteMessage(chatID, messageID)
				delBot := tgbotapi.NewDeleteMessage(chatID, sendMsg.MessageID)
				bot.Request(delUser)
				bot.Request(delBot)
				return
			}
			ind, ok := getUserState(chatID)
			if ok && ind["password_hash"] == "yes" {
				delUser := tgbotapi.NewDeleteMessage(chatID, messageID)
				delBot := tgbotapi.NewDeleteMessage(chatID, sendMsg.MessageID)
				bot.Request(delUser)
				bot.Request(delBot)
				setUserState(chatID, map[string]string{"password_hash": hashString})
				setReady(chatID, true)
				log.Printf("hP: password successfully set")
				return
			}
			if ok && ind["password_hash"] == "no" {
				delUser := tgbotapi.NewDeleteMessage(chatID, messageID)
				delBot := tgbotapi.NewDeleteMessage(chatID, sendMsg.MessageID)
				bot.Request(delUser)
				bot.Request(delBot)
				sendErrorMessage(bot, chatID, "Введите верный пароль", "", "")
				setReady(chatID, false)
				log.Printf("hP: password rejected")
				setUserState(chatID, map[string]string{"password": ""})
				return
			}
			if !ok {
				delUser := tgbotapi.NewDeleteMessage(chatID, messageID)
				delBot := tgbotapi.NewDeleteMessage(chatID, sendMsg.MessageID)
				bot.Request(delUser)
				bot.Request(delBot)
				deleteUserState(chatID)
				sendErrorMessage(bot, chatID, "error when accessing map", "", "")
				return
			}
			runtime.Gosched()
		}
	}()

	select {
	case <-timer.C:
		log.Printf("hP: password input timed out")
		return
	case <-done:
		timer.Stop()
		return
	}
}

func RootAndAdminHandler(bot *tgbotapi.BotAPI, chatID int64, from *tgbotapi.User, change bool) string {
	log.Printf("rAAH: started RootAndAdminHandler")
	timer := time.NewTimer(1 * time.Minute)
	tiker := time.NewTicker(250 * time.Millisecond)
	done := make(chan struct{})
	hashPasChan := make(chan string, 1)

	go func() {
		defer close(done)
		tik := 0
		defer close(hashPasChan)
		setUserState(chatID, map[string]string{"password": ""})
		if !change {
			sendSuccessMessage(bot, chatID,
				fmt.Sprintf(
					"Добро пожаловать в команду администраторов проекта %s, %s.\nВведите пароль, обязательно сохраните его вне Telegram.\nОн может содержать только - латиницу, цифры и символы '_', '#', '@', '-'",
					bot.Self.UserName,
					strings.TrimSpace(from.LastName+" "+from.FirstName),
				), "", "",
			)
		} else {
			sendSuccessMessage(bot, chatID,
				fmt.Sprintf(
					"%s измените свой пароль на новый.\nВведите новый пароль, обязательно сохраните его вне Telegram.\nОн может содержать только - латиницу, цифры и символы '_', '#', '@', '-'",
					strings.TrimSpace(from.LastName+" "+from.FirstName),
				), "", "",
			)
		}

		for range tiker.C {
			tik++
			if tik >= 240 {
				tik = 0
				return
			}
			if isReady(chatID) { // Проверяем готовность через userReady
				setReady(chatID, false) // Сбрасываем флаг готовности
				state, ok := getUserState(chatID)
				if ok {
					hashPasChan <- state["password_hash"]
					log.Printf("rAAH: password hash set")
				}
				return
			}
			runtime.Gosched()
		}
	}()

	select {
	case <-timer.C:
		log.Printf("rAAH: context timeout")
		return ""
	case <-done:
		timer.Stop()
		return <-hashPasChan
	}
}

func handelerPasswordChange(bot *tgbotapi.BotAPI, userID, chatID int64, old_password, status string, messageID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, hash_password, err := pg_us_db.GetUserStatus(ctx, userID)
	if err != nil {
		log.Printf("hPC: error whem change password: %v", err)
		sendErrorMessage(bot, chatID, "error whem change password, try again later", status, "")
		setUserState(chatID, map[string]string{"re_password": "false"})
		return
	}

	hash := sha256.Sum256([]byte(old_password))
	if hash_password == hex.EncodeToString(hash[:]) {
		setUserState(chatID, map[string]string{"re_password": "true"})
		delUser := tgbotapi.NewDeleteMessage(chatID, messageID)
		bot.Request(delUser)
		return
	} else {
		setUserState(chatID, map[string]string{"re_password": "false"})
		delUser := tgbotapi.NewDeleteMessage(chatID, messageID)
		bot.Request(delUser)
		return
	}
}

func isValid(s string) (rune, bool) {
	for _, ch := range s {
		if !isLatinLetter(ch) && !unicode.IsDigit(ch) && ch != '_' && ch != '#' && ch != '@' && ch != '-' {
			log.Printf("iV: Invalid character found: %c", ch)
			return ch, false
		}
	}
	return 0, true
}

func isLatinLetter(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func escapeMarkdownV2(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(s)
}
