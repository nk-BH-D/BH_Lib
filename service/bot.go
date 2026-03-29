package tg

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"

	//"os"
	//"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nk-BH-D/BH_Lib/internal/config"
	lib "github.com/nk-BH-D/BH_Lib/internal/method_lib_db"
	us "github.com/nk-BH-D/BH_Lib/internal/method_users_db"
)

var (
	userStates    = make(map[int64]map[string]string)
	sessionStatus = make(map[int64]sessionState)
	userReady     = make(map[int64]bool) // Карта для хранения состояния готовности
	decisionUsers = make(map[int64]chan string)
	uSM           sync.RWMutex
	sSM           sync.RWMutex
	uRM           sync.RWMutex
	dM            sync.RWMutex
	conf          *config.Config
	pg_lib_db     *lib.PostgresLib
	pg_us_db      *us.PostgresUs
	pg_ses_db     *us.PostgresSes
)

type sessionState struct {
	state        bool
	time_created time.Time
}

// func for getting config from main
func Init(cfg *config.Config, pg_lib *lib.PostgresLib, pg_us *us.PostgresUs, pg_ses *us.PostgresSes) {
	conf = cfg
	pg_lib_db = pg_lib
	pg_us_db = pg_us
	pg_ses_db = pg_ses
}

// func fot syncing userState map
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

// func for syncing statusState map
func setSessionState(userID int64, state bool) {
	sSM.Lock()
	defer sSM.Unlock()
	sessionStatus[userID] = sessionState{
		state:        state,
		time_created: time.Now().UTC(),
	}
}
func getSessionState(userID int64) (sessionState, bool) {
	sSM.RLock()
	defer sSM.RUnlock()
	state, ok := sessionStatus[userID]
	return state, ok
}
func deleteSessionStatus(userID int64) {
	sSM.Lock()
	defer sSM.Unlock()
	delete(sessionStatus, userID)
}

// func for syncing ready map
func setReady(chatID int64, ready bool) {
	uRM.Lock()
	defer uRM.Unlock()
	userReady[chatID] = ready
	log.Printf("setReady: флаг готовности установлен для chatID %d: %v", chatID, ready)
}
func isReady(chatID int64) bool {
	uRM.RLock()
	defer uRM.RUnlock()
	ready, ok := userReady[chatID]
	if !ok {
		return false
	}
	return ready
}

// func for syncing decision map
func setDecisionUsers(chatID int64, decision string) {
	dM.Lock()
	defer dM.Unlock()

	if _, ok := decisionUsers[chatID]; !ok {
		decisionUsers[chatID] = make(chan string, 1) // буфер 1
	}

	// если канал уже занят, перезаписываем
	select {
	case decisionUsers[chatID] <- decision:
	default:
		// канал переполнен, переписываем старое сообщение
		<-decisionUsers[chatID]
		decisionUsers[chatID] <- decision
	}
}
func getDecisionUsers(chatID int64) string {
	dM.RLock()
	defer dM.RUnlock()

	if ch, ok := decisionUsers[chatID]; ok {
		select {
		case msg := <-ch:
			return msg
		default:
			return "" // нет сообщения
		}
	}

	return "" // канал не найден
}

// sendDocxFile sending file .docx to Telegram chat.
//func sendDocxFile(bot *tgbotapi.BotAPI, chatID int64, filePath string, status string) {
//	if _, err := os.Stat(filePath); os.IsNotExist(err) {
//		log.Printf("file '%s' not found: %v", filePath, err)
//		return
//	}
//
//	fileBytes, err := os.ReadFile(filePath)
//	if err != nil {
//		log.Printf("couldn't read the file '%s': %v", filePath, err)
//		return
//	}
//
//	fileName := filepath.Base(filePath)
//	// creating an object DocumentConfig for sending to Telegram
//	documentConfig := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
//		Name:  fileName,
//		Bytes: fileBytes,
//	})
//
//	documentConfig.Caption = "Внимательно изучите содержимое файла"
//	switch status {
//	case "root":
//		documentConfig.ReplyMarkup = createRootMenuKeyboard()
//	case "admin":
//		documentConfig.ReplyMarkup = createAdminMenuKeyboard()
//	case "user":
//		documentConfig.ReplyMarkup = createUserMenuKeyboard()
//	default:
//		msg := tgbotapi.NewMessage(
//			chatID,
//			"Internal service error: попробуйте снова позже",
//		)
//		bot.Send(msg)
//		log.Printf("error whem detect user status: %s", status)
//		return
//	}
//
//	_, err = bot.Send(documentConfig)
//	if err != nil {
//		log.Printf("couldn't send the file '%s' to chat %d: %v", filePath, chatID, err)
//		return
//	}
//}

func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("НM %+v", message.From)
	chatID := message.Chat.ID
	userID := message.From.ID
	messageID := message.MessageID
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
		sendSuccessMessage(
			bot,
			chatID,
			"Из-за замедления Telegram файлы плохо или вообще не загружаютсяю\nПоэтому вы можете ознакомится с кодом и инструкцией по использованию тут.\nGitHub: https://github.com/nk-BH-D/BH_Lib ",
			status,
			"",
		)
		//switch status {
		//case "root":
		//	sendDocxFile(bot, chatID, conf.DOCX_ROOT_PATH, status)
		//case "admin":
		//	sendDocxFile(bot, chatID, conf.DOCX_ADMIN_PATH, status)
		//case "user":
		//	sendDocxFile(bot, chatID, conf.DOCX_USER_PATH, status)
		//default:
		//	msg := tgbotapi.NewMessage(
		//		chatID,
		//		"Internal service error: попробуйте снова позже 8",
		//	)
		//	bot.Send(msg)
		//}
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
				go handlePassword(bot, chatID, text, messageID)
				log.Printf("Активных горутин: %d\n", runtime.NumGoroutine())
				return
			case "re_password":
				go handelerPasswordChange(bot, userID, chatID, text, status, messageID)
				log.Printf("Активных горутин: %d\n", runtime.NumGoroutine())
				return
			case "password_check":
				setUserState(chatID, map[string]string{"password_check": text})
				delUser := tgbotapi.NewDeleteMessage(chatID, messageID)
				bot.Request(delUser)
				setReady(chatID, true)
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
	message_list := []string{}
	if len(message) > 4096 {
		runes := []rune(message)
		for i := 0; i < len(runes); i += 4096 {
			end := i + 4096

			if end > len(runes) {
				end = len(runes) // хвост
			}

			message_list = append(message_list, string(runes[i:end]))
		}
		assistedSendSuccessMessage(bot, chatID, message_list, status, mod)
	} else {
		message_list = append(message_list, message)
		assistedSendSuccessMessage(bot, chatID, message_list, status, mod)
	}
}
func assistedSendSuccessMessage(bot *tgbotapi.BotAPI, chatID int64, message_list []string, status, mod string) {
	for _, message := range message_list {
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

		if mod == "md" {
			msg.ParseMode = "MarkdownV2"
		}

		_, sendErr := bot.Send(msg)
		if sendErr != nil {
			log.Printf("error sending message: %s, %v", message, sendErr)
			sendErrorMessage(bot, chatID, fmt.Sprintf("произошла ошибка при отправке сообщения: %v", sendErr), status, "")
			log.Printf("ошибка при отправкее сообщения: %v", sendErr)
			return
		}
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
		if chekerSessionStatus(bot, chatID, userID, status) {
			setUserState(chatID, map[string]string{"create_code": ""})
			sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
			return
		}
		return
	case "get_code":
		setUserState(chatID, map[string]string{"get_code": ""})
		sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
		return
	case "del_code":
		if chekerSessionStatus(bot, chatID, userID, status) {
			setUserState(chatID, map[string]string{"del_code": ""})
			sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
			return
		}
		return
	case "up_code":
		if chekerSessionStatus(bot, chatID, userID, status) {
			setUserState(chatID, map[string]string{"up_code": ""})
			sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
			return
		}
		return
	case "get_my_request":
		go handlerGetRequest(bot, chatID, userID, status)
		log.Printf("Активныч горутин: %d\n", runtime.NumGoroutine())
		return
	case "new_admin":
		if chekerSessionStatus(bot, chatID, userID, status) {
			setUserState(chatID, map[string]string{"new_admin": ""})
			sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
			return
		}
		return
	case "del_admin":
		if chekerSessionStatus(bot, chatID, userID, status) {
			setUserState(chatID, map[string]string{"del_admin": ""})
			sendSuccessMessage(bot, chatID, "Введите данные как указано в документации", status, "d")
			return
		}
		return
	case "decision":
		result := getDecisionUsers(chatID)
		sendSuccessMessage(bot, chatID, result, status, "md")
		return
	default:
		sendErrorMessage(bot, chatID, "Неизвестная команда", status, "")
		return
	}
}

func chekerSessionStatus(bot *tgbotapi.BotAPI, chatID, userID int64, status string) bool {
	timer := time.NewTimer(1 * time.Minute)
	tiker := time.NewTicker(250 * time.Millisecond)
	done := make(chan struct{})
	res_chan := make(chan bool, 1)

	go func() {
		defer close(done)
		tik := 0
		state, ok := getSessionState(userID)
		if !ok {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, ind, err := pg_ses_db.GetSession(ctx, userID)
			if err != nil {
				log.Printf("error whem getting state for session: %v", err)
				sendErrorMessage(bot, chatID, "Internal service error 34", status, "")
				res_chan <- false
				return
			}
			if ind {
				setSessionState(userID, false)
				setUserState(chatID, map[string]string{"password_check": ""})
				sendSuccessMessage(bot, chatID, "Введите пароль", status, "")
				for range tiker.C {
					tik++
					if tik >= 240 {
						tik = 0
						return
					}
					if isReady(chatID) {
						setReady(chatID, false)
						ctxGUS, cancelGUS := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancelGUS()
						_, _, password_in_db, err := pg_us_db.GetUserStatus(ctxGUS, userID)
						if err != nil {
							log.Printf("error whem getting password in db")
							sendErrorMessage(bot, chatID, "Internal service error 36", status, "")
							res_chan <- false
							return
						}
						password, ok := getUserState(chatID)
						if !ok {
							log.Println("error whem getting user status from db")
							sendErrorMessage(bot, chatID, "Internal service error 35", status, "")
							res_chan <- false
							return
						}
						hash := sha256.Sum256([]byte(password["password_check"]))
						if password_in_db == hex.EncodeToString(hash[:]) {
							setSessionState(userID, true)
							sendSuccessMessage(bot, chatID, "Пароль принят, у вас есть 15 минут до окончания сессии", "", "")
							res_chan <- true
							return
						} else {
							sendErrorMessage(bot, chatID, "Пароль не верный, повторно выберите действие", status, "")
							res_chan <- false
							return
						}
					}
					runtime.Gosched()
				}
			}
		}
		if state.state {
			if time.Since(state.time_created) > 15*time.Minute {
				setSessionState(userID, false)
				setUserState(chatID, map[string]string{"password_check": ""})
				sendSuccessMessage(bot, chatID, "Введите пароль", status, "")
				for range tiker.C {
					tik++
					if tik >= 240 {
						tik = 0
						return
					}
					if isReady(chatID) {
						setReady(chatID, false)
						ctxGUS, cancelGUS := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancelGUS()
						_, _, password_in_db, err := pg_us_db.GetUserStatus(ctxGUS, userID)
						if err != nil {
							log.Printf("error whem getting password in db")
							sendErrorMessage(bot, chatID, "Internal service error 36", status, "")
							res_chan <- false
							return
						}
						password, ok := getUserState(chatID)
						if !ok {
							log.Println("error whem getting user status from db")
							sendErrorMessage(bot, chatID, "Internal service error 35", status, "")
							res_chan <- false
							return
						}
						hash := sha256.Sum256([]byte(password["password_check"]))
						if password_in_db == hex.EncodeToString(hash[:]) {
							setSessionState(userID, true)
							sendSuccessMessage(bot, chatID, "Пароль принят, у вас есть 15 минут до окончания сессии", "", "")
							res_chan <- true
							return
						} else {
							sendErrorMessage(bot, chatID, "Пароль не верный, повторно выберите действие", status, "")
							res_chan <- false
							return
						}
					}
					runtime.Gosched()
				}
			} else {
				remaining := (15 * time.Minute) - time.Since(state.time_created)
				minute := int(remaining.Minutes())
				second := int(remaining.Seconds()) % 60
				sendSuccessMessage(bot, chatID, fmt.Sprintf(
					"У вас осталось %d минут %d секунд", minute, second,
				), "", "",
				)
				res_chan <- true
				return
			}
		}
		setUserState(chatID, map[string]string{"password_check": ""})
		sendSuccessMessage(bot, chatID, "Введите пароль", status, "")
		for range tiker.C {
			tik++
			if tik >= 240 {
				tik = 0
				return
			}
			if isReady(chatID) {
				setReady(chatID, false)
				ctxGUS, cancelGUS := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancelGUS()
				_, _, password_in_db, err := pg_us_db.GetUserStatus(ctxGUS, userID)
				if err != nil {
					log.Printf("error whem getting password in db")
					sendErrorMessage(bot, chatID, "Internal service error 36", status, "")
					res_chan <- false
					return
				}
				password, ok := getUserState(chatID)
				if !ok {
					log.Println("error whem getting user status from db")
					sendErrorMessage(bot, chatID, "Internal service error 35", status, "")
					res_chan <- false
					return
				}
				hash := sha256.Sum256([]byte(password["password_check"]))
				if password_in_db == hex.EncodeToString(hash[:]) {
					setSessionState(userID, true)
					sendSuccessMessage(bot, chatID, "Пароль принят, у вас есть 15 минут до окончания сессии", "", "")
					res_chan <- true
					return
				} else {
					sendErrorMessage(bot, chatID, "Пароль не верный, повторно выберите действие", status, "")
					res_chan <- false
					return
				}
			}
			runtime.Gosched()
		}
	}()

	select {
	case <-timer.C:
		log.Println("cSS: password input timed out")
		sendErrorMessage(bot, chatID, "Сессия ввода пароля истекла, повторно выберите действие", status, "")
		return false
	case <-done:
		timer.Stop()
		res := <-res_chan
		return res
	}
}
