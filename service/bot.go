package tg

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	//"github.com/google/uuid"
	"github.com/nk-BH-D/BH_Lib/internal/config"
	lib "github.com/nk-BH-D/BH_Lib/internal/method_lib_db"
	us "github.com/nk-BH-D/BH_Lib/internal/method_users_db"
)

var (
	userStates      = make(map[int64]map[string]string)
	ready      bool = false
	uSM        sync.RWMutex
	conf       *config.Config
	pg_lib_db  *lib.PostgresLib
	pg_us_db   *us.PostgresUs
)

// func for getting config from main
func Init(cfg *config.Config, pg_lib *lib.PostgresLib, pg_us *us.PostgresUs) {
	conf = cfg
	pg_lib_db = pg_lib
	pg_us_db = pg_us
}

// func fot syncing map

func setUserState(chatID int64, state map[string]string) {
	uSM.Lock()
	defer uSM.Unlock()
	userStates[chatID] = state
}
func getUserState(chatID int64) (map[string]string, bool) {
	uSM.RLock()
	defer uSM.RUnlock()
	state, ok := userStates[chatID]
	return state, ok
}
func deleteUserState(chatID int64) {
	uSM.Lock()
	defer uSM.Unlock()
	delete(userStates, chatID)
}

// sendDocxFile sending file .docx to Telegram chat.
func sendDocxFile(bot *tgbotapi.BotAPI, chatID int64, filePath string, status string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("file '%s' not found: %v", filePath, err)
		return
	}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("couldn't read the file '%s': %v", filePath, err)
		return
	}

	fileName := filepath.Base(filePath)
	// creating an object DocumentConfig for sending to Telegram
	documentConfig := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
		Name:  fileName,
		Bytes: fileBytes,
	})

	documentConfig.Caption = "Внимательно изучите содержимое файла"
	switch status {
	case "root":
		documentConfig.ReplyMarkup = createRootMenuKeyboard()
	case "admin":
		documentConfig.ReplyMarkup = createAdminMenuKeyboard()
	case "user":
		documentConfig.ReplyMarkup = createUserMenuKeyboard()
	default:
		msg := tgbotapi.NewMessage(
			chatID,
			"Internal service error: попробуйте снова позже",
		)
		bot.Send(msg)
		log.Printf("error whem detect user status: %s", status)
		return
	}

	_, err = bot.Send(documentConfig)
	if err != nil {
		log.Printf("couldn't send the file '%s' to chat %d: %v", filePath, chatID, err)
		return
	}
}

func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("НM %+v", message.From)
	chatID := message.Chat.ID
	userID := message.From.ID
	text := strings.TrimSpace(message.Text)
	//log.Printf("HM text: %s", text)
	//log.Printf("HM message: %v", message.Text)
	login := message.From.UserName

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, status, _, err := pg_us_db.GetUserStatus(ctx, userID)
	if err != nil && err != sql.ErrNoRows {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 12", status, "")
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Intercepted panic: %v", r)
			sendErrorMessage(bot, chatID, "Что-то пошло не так, попробуйте снова", status, "p")
			return
		}
	}()

	if text == "/docx" {
		deleteUserState(chatID)
		switch status {
		case "root":
			sendDocxFile(bot, chatID, conf.DOCX_ROOT_PATH, status)
		case "admin":
			sendDocxFile(bot, chatID, conf.DOCX_ADMIN_PATH, status)
		case "user":
			sendDocxFile(bot, chatID, conf.DOCX_USER_PATH, status)
		default:
			msg := tgbotapi.NewMessage(
				chatID,
				"Internal service error: попробуйте снова позже 8",
			)
			bot.Send(msg)
		}
		return
	}

	if text == "/start" {
		deleteUserState(chatID)
		go showMainMenu(bot, chatID, message.From)
		log.Printf("Активныч горутин: %d\n", runtime.NumGoroutine())
		return
	}
	// processing the bot's status
	if state, ok := getUserState(chatID); ok {
		for key := range state {
			switch key {
			case "get_code":
				go handlerGetCod(bot, chatID, userID, status, text, login)
				log.Printf("Активныч горутин: %d\n", runtime.NumGoroutine())
				return
			case "create_code":
				go handlerCreateCod(bot, chatID, userID, status, message.Text, login)
				log.Printf("Активныч горутин: %d\n", runtime.NumGoroutine())
				return
			case "password":
				go handlePassword(bot, chatID, text)
				log.Printf("Активных горутин: %d\n", runtime.NumGoroutine())
				return
			case "re_password":
				go handelerPasswordChange(bot, userID, chatID, text, status)
				log.Printf("Активных горутин: %d\n", runtime.NumGoroutine())
				return
			case "del_code":
				go handlerDelCode(bot, chatID, userID, status, message.Text, message.From.UserName)
				log.Printf("Активных горутин: %d\n", runtime.NumGoroutine())
				return
			case "up_code":
				go handlerUpCode(bot, chatID, userID, status, message.Text, message.From.UserName)
				log.Printf("Активных горутин: %d\n", runtime.NumGoroutine())
				return
			case "new_admin":
				go handlerNewAdmin(bot, chatID, userID, status, text)
				log.Printf("Активныч горутин: %d\n", runtime.NumGoroutine())
				return
			case "del_admin":
				go handlerDelAdmin(bot, chatID, userID, status, text)
				log.Printf("Активныч горутин: %d\n", runtime.NumGoroutine())
				return
			default:
				sendErrorMessage(bot, chatID, "Неизвестная команда", status, "")
				return
			}
		}
	}
}

// rk - RootCommands
// "" - internal error or defalt
// ak - AdminCammands
// uk - UserCommands
// p - panic
// cp - ChekPassword and MarcdownV2
// d - вызовет internal error на случай если status по какой-то причине не сработает

func sendErrorMessage(bot *tgbotapi.BotAPI, chatID int64, message string, status, mod string) {
	msg := tgbotapi.NewMessage(chatID, message)
	switch status {
	case "root":
		msg.ReplyMarkup = createRootMenuKeyboard()
	case "admin":
		msg.ReplyMarkup = createAdminMenuKeyboard()
	case "user":
		msg.ReplyMarkup = createUserMenuKeyboard()
	default:
		switch mod {
		case "":
			msg.ReplyMarkup = nil
		case "p":
			msg.ReplyMarkup = nil
		default:
			sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 11", "", "")
			return
		}

	}
	_, sendErr := bot.Send(msg)
	if sendErr != nil {
		log.Printf("error sending message: %s, %v", message, sendErr)
		return
	}
}
func sendSuccessMessage(bot *tgbotapi.BotAPI, chatID int64, message string, status, mod string) {
	msg := tgbotapi.NewMessage(chatID, message)
	switch status {
	case "root":
		msg.ReplyMarkup = createRootMenuKeyboard()
	case "admin":
		msg.ReplyMarkup = createAdminMenuKeyboard()
	case "user":
		msg.ReplyMarkup = createUserMenuKeyboard()
	default:
		switch mod {
		case "rk":
			msg.ReplyMarkup = createRootCommandsKeyboard()
		case "ak":
			msg.ReplyMarkup = createAdminCommandsKeyboard()
		case "uk":
			msg.ReplyMarkup = createUserCommandsKeyboard()
		case "":
			msg.ReplyMarkup = nil
		case "cp":
			msg.ReplyMarkup = createCheckPasswordKeybord()
			msg.ParseMode = "MarkdownV2"
		default:
			sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 13", "", "")
			log.Printf("ошибка при определении status или mod")
			return
		}

	}
	_, sendErr := bot.Send(msg)
	if sendErr != nil {
		log.Printf("error sending message: %s, %v", message, sendErr)
		sendErrorMessage(bot, chatID, fmt.Sprintf("произошла ошибка при отправке сообщения: %v", sendErr), status, "")
		log.Printf("ошибка при отправкее сообщения: %v", sendErr)
		return
	}
}

// frontend
func HandleCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	log.Printf("НC %+v", callbackQuery.From)
	chatID := callbackQuery.Message.Chat.ID
	data := callbackQuery.Data
	userID := callbackQuery.From.ID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, status, _, err := pg_us_db.GetUserStatus(ctx, callbackQuery.From.ID)
	if err != nil && err != sql.ErrNoRows {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 10", status, "")
		log.Printf("HC: error whem GetUserStatus: %v", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Перехвачена паника: %v", r)
			sendErrorMessage(bot, chatID, "Что-то пошло не так, попробуйте снова", status, "p")
			return
		}
	}()
	log.Printf("HC data: %s", data)
	switch data {
	case "show_root_menu":
		deleteUserState(chatID)
		if status != "root" {
			sendErrorMessage(bot, chatID, "Вам отказано в доступе", status, "")
			return
		} else {
			sendSuccessMessage(bot, chatID, "Выберите действие", "", "rk")
			return
		}
	case "show_admin_menu":
		deleteUserState(chatID)
		if status != "admin" {
			sendErrorMessage(bot, chatID, "Вам отказано в доступе", "user", "")
			return
		} else {
			sendSuccessMessage(bot, chatID, "Выберите действие", "", "ak")
			return
		}
	case "show_user_menu":
		deleteUserState(chatID)
		sendSuccessMessage(bot, chatID, "Выберите действие", "", "uk")
		return
	case "yes":
		setUserState(chatID, map[string]string{"password_hash": "yes"})
		return
	case "no":
		setUserState(chatID, map[string]string{"password_hash": "no"})
		return
	case "create_code":
		setUserState(chatID, map[string]string{"create_code": ""})
		sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
		return
	case "get_code":
		setUserState(chatID, map[string]string{"get_code": ""})
		sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
		return
	case "del_code":
		setUserState(chatID, map[string]string{"del_code": ""})
		sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
		return
	case "up_code":
		setUserState(chatID, map[string]string{"up_code": ""})
		sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
		return
	case "get_my_request":
		go handlerGetRequest(bot, chatID, userID, status)
		log.Printf("Активныч горутин: %d\n", runtime.NumGoroutine())
		return
	case "new_admin":
		setUserState(chatID, map[string]string{"new_admin": ""})
		sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
		return
	case "del_admin":
		setUserState(chatID, map[string]string{"del_admin": ""})
		sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
		return
	default:
		sendErrorMessage(bot, chatID, "Неизвестная команда", status, "")
		return
	}
}
