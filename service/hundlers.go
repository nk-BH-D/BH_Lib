package tg

import (
	"context"
	//"crypto/sha256"
	"database/sql"
	//"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func handlerCreateCod(bot *tgbotapi.BotAPI, chatID, userID int64, status, message, login string) {
	//log.Println("запустились")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	login_in_db, status_in_db, _, err := pg_us_db.GetUserStatus(ctx, userID)
	//log.Printf("статус: %s %s; err: %v", status_in_db, status, err)
	if err != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 10", status, "")
		log.Printf("hCC: error whem GetUserStatus: %v", err)
		return
	}

	if login_in_db != login {
		ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel1()
		err := pg_us_db.UpdateUserData(ctx1, userID, login, "", 0)
		if err != nil {
			log.Printf("error whem up user status: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error 32", status, "")
			return
		}
	}

	if status_in_db == status && status_in_db != "user" {
		parts := strings.Split(message, ";;")
		log.Printf("len: %d", len(parts))

		if len(parts) < 4 {
			sendErrorMessage(bot, chatID, "Данные введены не корректно", status, "")
			log.Println("hCC: len parts != 4")
			return
		}

		course_name := strings.TrimSpace(parts[0])

		for i := 1; i+2 <= len(parts); i += 3 {
			task_name := strings.TrimSpace(parts[i])
			cound := strings.TrimSpace(parts[i+1])
			decision := parts[i+2]
			//log.Println(cound)
			//log.Println(parts[i+2])

			ctxIN, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := pg_lib_db.InsertData(ctxIN, userID, course_name, task_name, cound, decision)
			log.Println(err)
			if err != nil {
				if pgErr, ok := err.(*pgconn.PgError); ok {
					if pgErr.Code == "23505" {
						sendErrorMessage(bot, chatID, "Решение этой задачи уже есть в базе", status, "")
						log.Printf("пользователь: %s попытался добавить решение задачи: %s/%s, которое уже есть в базе", login, parts[0], parts[i])
						return
					}
				}
				sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 15", status, "")
				log.Printf("hCC: error whem InsertData: %v", err)
				return
			}

			ctxGUR, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel1()
			rq, _, err := pg_us_db.GetUserRequestnames(ctxGUR, userID)
			if err != nil {
				sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 15", status, "")
				log.Printf("hCC: error whem GetUserRequestnames: %v", err)
				return
			}

			if !strings.Contains(rq, "c"+"/"+course_name+"/"+task_name) && !strings.Contains(rq, "/"+course_name+"/"+task_name) {
				ctxUD, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel2()
				errUD := pg_us_db.UpdateUserData(ctxUD, userID, login, "c"+"/"+course_name+"/"+task_name, 1)
				if errUD != nil {
					sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 17", status, "")
					log.Printf("hCC: error whem UpdateUserData: %v", errUD)
					return
				}
			}
			sendSuccessMessage(bot, chatID, fmt.Sprintf("Решение: '%s' успешно сохранено", task_name), status, "")
		}
	} else {
		sendErrorMessage(bot, chatID, "Вам отказано в доступе", status, "")
		log.Printf("hCC: access denied for user %d", userID)
		return
	}
}

func handlerGetCod(bot *tgbotapi.BotAPI, chatID, userID int64, status, message, login string) {
	ctx0, cancel0 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel0()
	login_in_db, _, _, err := pg_us_db.GetUserStatus(ctx0, userID)
	if err != nil {
		log.Printf("error whem get user status: %v", err)
		sendErrorMessage(bot, chatID, "Internal service error 30", status, "")
		return
	}

	if login_in_db != login {
		ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel1()
		err := pg_us_db.UpdateUserData(ctx1, userID, login, "", 0)
		if err != nil {
			log.Printf("error whem up user status: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error 31", status, "")
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := strings.ReplaceAll(message, "\r", "")
	parts := strings.Split(res, "\n")
	//name_course_rq = parts[0]
	//name_task_rq = parts[1]

	if len(parts) != 2 {
		sendErrorMessage(bot, chatID, "Данные не коректны", status, "")
		log.Println("hGC: data incorrect")
		return
	}

	name, cond, data, err := pg_lib_db.GetData(ctx, parts[0], parts[1])
	if err != nil && err != sql.ErrNoRows {
		sendErrorMessage(bot, chatID, "Internal service error, попробуйте позже 17", status, "")
		log.Printf("hGC: error whem GetData: %v", err)
		return
	}

	if err == sql.ErrNoRows {
		sendErrorMessage(bot, chatID, "Решения такой задачи нету в базе", status, "")
		log.Printf("hGC: no data found for parts: %s, %s", parts[0], parts[1])
		return
	}

	ctxGUR, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()
	rq, _, err := pg_us_db.GetUserRequestnames(ctxGUR, userID)
	if err != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 15", status, "")
		log.Printf("hCC: error whem GetUserRequestnames: %v", err)
		return
	}

	if !strings.Contains(rq, "c"+"/"+parts[0]+"/"+parts[1]) && !strings.Contains(rq, "/"+parts[0]+"/"+parts[1]) {
		ctxUD, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel1()
		errUD := pg_us_db.UpdateUserData(ctxUD, userID, login, "/"+parts[0]+"/"+parts[1], 1)
		if errUD != nil {
			sendErrorMessage(bot, chatID, "Internal service error, попробуйте позже 18", status, "")
			log.Printf("hGC: error whem UpdateUserData: %v", errUD)
			return
		}
	}

	var result strings.Builder

	if cond == "" {
		result.WriteString(escapeMarkdownV2(fmt.Sprintf("Задача: %s\n", name)))
		result.WriteString("```" + "\n")
		result.WriteString(escapeMarkdownV2(data) + "\n")
		result.WriteString("```" + "\n")
	} else {
		result.WriteString(escapeMarkdownV2(fmt.Sprintf("Задача: %s\n", name)))
		result.WriteString("\n")
		result.WriteString("Условие\n" + escapeMarkdownV2(cond) + "\n")
		result.WriteString("\n")
		result.WriteString("```" + "\n")
		result.WriteString(escapeMarkdownV2(data) + "\n")
		result.WriteString("```" + "\n")
	}

	msg := tgbotapi.NewMessage(chatID, result.String())

	log.Printf("hGC: status: %s", status)
	switch status {
	case "root":
		msg.ReplyMarkup = createRootMenuKeyboard()
	case "admin":
		msg.ReplyMarkup = createAdminMenuKeyboard()
	case "user":
		msg.ReplyMarkup = createUserMenuKeyboard()
	default:
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 25", "", "")
		log.Printf("hGC: error whem detect user status: %s", status)
		return
	}
	msg.ParseMode = "MarkdownV2"
	_, errSend := bot.Send(msg)
	if errSend != nil {
		sendErrorMessage(bot, chatID, "error whem sending message", status, "")
		log.Printf("error whem sending message: %s, err: %v", result.String(), errSend)
		return
	}
}

func handlerDelCode(bot *tgbotapi.BotAPI, chatID, userID int64, status, message, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, status_in_db, _, err := pg_us_db.GetUserStatus(ctx, userID)
	if err != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 10", status, "")
		log.Printf("hCC: error whem GetUserStatus: %v", err)
		return
	}

	if status_in_db == status && status != "user" {
		parts := strings.Split(message, ";;")
		if len(parts) != 4 {
			sendErrorMessage(bot, chatID, "Данные введены не корректно", status, "")
			log.Println("hDC: len parts != 4")
			return
		}

		format := strings.ReplaceAll("/"+parts[0]+"/"+parts[1], "\n", "")
		format = strings.ReplaceAll(format, "\r", "")

		ctxG, cancelG := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelG()
		rqn, _, err := pg_us_db.GetUserRequestnames(ctxG, userID)
		if err != nil {
			sendErrorMessage(bot, chatID, "Internal service error 28", status, "")
			log.Printf("hDC error whem GetUserRequestName: %v", err)
			return
		}
		valid := false
		frqn := strings.ReplaceAll(rqn, "\r", "")
		chek_parts := strings.Split(frqn, "\n")
		for _, str := range chek_parts {
			if "up"+format == str || "c"+format == str {
				valid = true
				break
			}
		}
		if !valid {
			sendErrorMessage(
				bot,
				chatID,
				"Решение, которое вы пытаетесь удалить было добавлено другим администратором. У вас нет прав на его удаление",
				status,
				"",
			)
			log.Printf("Отказанно в удалении решения: %s пользователю: %s", format, name)
			return
		}

		ctxD, cancelD := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelD()
		errD := pg_lib_db.DeleteData(ctxD, parts[2], userID)
		if errD != nil {
			sendErrorMessage(bot, chatID, "Ошибка при удалении данных", status, "")
			log.Printf("error whem del datat: %v", errD)
			return
		}
		ctxDR, cancelDR := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelDR()
		errDR := pg_us_db.DeleteReqData(ctxDR, format, userID)
		log.Printf("hDC: %s", format)
		if errDR != nil {
			sendErrorMessage(bot, chatID, "Ошибка при удалении данных из списка запросов", status, "")
			log.Printf("error whem del data: %v", errDR)
			return
		}
		sendSuccessMessage(bot, chatID, "Решение успешно удалено", status, "")
		return
	} else {
		sendErrorMessage(bot, chatID, "Вам отказано в доступе", status, "")
		log.Printf("hDC: access denied for user %d", userID)
		return
	}
}

func handlerUpCode(bot *tgbotapi.BotAPI, chatID, userID int64, status, message, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, status_in_db, _, err := pg_us_db.GetUserStatus(ctx, userID)
	if err != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 10", status, "")
		log.Printf("hCC: error whem GetUserStatus: %v", err)
		return
	}

	if status_in_db == status && status != "user" {
		parts := strings.Split(message, ";;")
		if len(parts) != 4 {
			sendErrorMessage(bot, chatID, "Данные введены не корректно", status, "")
			log.Println("hDC: len parts != 4")
			return
		}

		format1 := strings.TrimSpace(parts[0])
		format2 := strings.TrimSpace(parts[1])
		cond := strings.TrimSpace(parts[2])

		ctxG, cancelG := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelG()
		rqn, _, err := pg_us_db.GetUserRequestnames(ctxG, userID)
		if err != nil {
			sendErrorMessage(bot, chatID, "Internal service error 28", status, "")
			log.Printf("hDC error whem GetUserRequestName: %v", err)
			return
		}
		valid := false
		frqn := strings.ReplaceAll(rqn, "\r", "")
		chek_parts := strings.Split(frqn, "\n")
		for _, str := range chek_parts {
			if "up"+"/"+format1+"/"+format2 == str || "c"+"/"+format1+"/"+format2 == str {
				valid = true
				break
			}
		}
		if !valid {
			sendErrorMessage(
				bot,
				chatID,
				"Решение, которое вы пытаетесь обновить было добавлено другим администратором. У вас нет прав на его обновление",
				status,
				"",
			)
			log.Printf("Отказанно в обновлении решения: %s пользователю: %s", "/"+format1+"/"+format2, name)
			return
		}

		ctxU, cancelD := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelD()
		errU := pg_lib_db.UpdateData(ctxU, format1, format2, cond, parts[3])
		if errU != nil {
			sendErrorMessage(bot, chatID, "Ошибка при удалении данных", status, "")
			log.Printf("error whem del datat: %v", errU)
			return
		}
		ctxUR, cancelDR := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelDR()
		errUR := pg_us_db.UpUserData(ctxUR, userID, "/"+format1+"/"+format2)
		if errUR != nil {
			sendErrorMessage(bot, chatID, "Ошибка при обновлении данных", status, "")
			log.Printf("error whem del datat: %v", errU)
			return
		}
		sendSuccessMessage(bot, chatID, "Решение успешно обновлено", status, "")
		return
	} else {
		sendErrorMessage(bot, chatID, "Вам отказано в доступе", status, "")
		log.Printf("hDC: access denied for user %d", userID)
		return
	}
}

func handlerGetRequest(bot *tgbotapi.BotAPI, chatID, userID int64, status string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if status == "root" || status == "admin" {
		result, err := pg_us_db.GetAdminAndRootRequests(ctx)
		if err != nil {
			log.Printf("error GAARR: %v", err)
			sendErrorMessage(bot, chatID, "ошибка GAARR", status, "")
			return
		}
		sendSuccessMessage(bot, chatID, result, status, "")
		return
	}
	if status == "user" {
		rn, q, err := pg_us_db.GetUserRequestnames(ctx, userID)
		if rn == "" || q == 0 {
			sendSuccessMessage(bot, chatID, "У вас пока нету запросов, начните пользоваться ботом что бы они появились", status, "")
			return
		}
		if err != nil {
			sendSuccessMessage(bot, chatID, "Internal service error, попробуйте позже 18", status, "")
			log.Printf("hGR: error whem GetUserRequestnames: %v", err)
			return
		}
		var result strings.Builder

		frn := strings.ReplaceAll(rn, "\r", "")
		parts := strings.Split(frn, "\n")

		for _, part := range parts {
			if part != "" {
				result.WriteString(part + "\n")
			}
		}

		result.WriteString(fmt.Sprintf("Общее количество запросов: %d", q))

		sendSuccessMessage(bot, chatID, result.String(), status, "")
		return
	}
}

func handlerNewAdmin(bot *tgbotapi.BotAPI, chatID, userID int64, status, message string) {
	log.Printf("hNA: started handlerNewAdmin")
	parts := strings.Split(message, " ")
	if len(parts) != 2 {
		sendErrorMessage(bot, chatID, "Данные введены не корректно", status, "")
		log.Printf("hNA: invalid input data")
		return
	}

	num_user_id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 19", status, "")
		log.Printf("hNA: error parsing user ID: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	login, _, _, err := pg_us_db.GetUserStatus(ctx, num_user_id)
	if err != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 18", status, "")
		log.Printf("hNA: error getting user status: %v", err)
		return
	}

	ctxR, cancelR := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelR()
	_, status_in_db, _, errR := pg_us_db.GetUserStatus(ctxR, userID)
	if errR != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 23", status, "")
		log.Printf("hNA: error getting root user status: %v", errR)
		return
	}

	if status_in_db == status && status == "root" {
		ctxCUS, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		errCUS := pg_us_db.ChangeUserStatus(ctxCUS, num_user_id, "admin", "")
		if errCUS != nil {
			sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 20", status, "")
			log.Printf("hNA: error changing user status: %v", errCUS)
			return
		}
		sendSuccessMessage(bot, chatID, fmt.Sprintf("Админ: %s успешно добавлен", login), status, "")
		intnACI, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 27", status, "")
			return
		}
		sendSuccessMessage(bot, intnACI, "Вы добавлены в список администраторов.\nНажмите /start что бы пройти регистрацию", "", "")
		log.Printf("hNA: admin %s successfully added", login)
		intUsID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			log.Printf("error whem parse parts[0]: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error 32", status, "")
			return
		}

		setSessionState(intUsID, false)
		ctxIS, closeIS := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeIS()
		errIS := pg_ses_db.InsertSession(ctxIS, intUsID, login)
		if errIS != nil {
			log.Printf("error whem insert session: %v", errIS)
			sendErrorMessage(bot, chatID, "Internal service error 33", status, "")
			return
		}

		ind, ok := getSessionState(intUsID)
		if !ok {
			log.Printf("error whem reading map: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error 33", status, "")
			return
		}
		log.Printf("hNA: admin sessiom created, status = %t, time = %v; for: %d", ind.state, ind.time_created, intUsID)
		return
	} else {
		sendErrorMessage(bot, chatID, "Вам отказано в доступе", status, "")
		log.Printf("hNA: access denied for user %d", userID)
		return
	}
}

func handlerDelAdmin(bot *tgbotapi.BotAPI, chatID, userID int64, status, message string) {
	log.Printf("hDA: started handlerDelAdmin")
	parts := strings.Split(message, " ")
	if len(parts) != 2 {
		sendErrorMessage(bot, chatID, "Данные введены не корректно", status, "")
		log.Printf("hDA: invalid input data")
		return
	}

	num_user_id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 21", status, "")
		log.Printf("hDA: error parsing user ID: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	login, _, _, err := pg_us_db.GetUserStatus(ctx, num_user_id)
	if err != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 22", status, "")
		log.Printf("hDA: error getting user status: %v", err)
		return
	}

	ctxR, cancelR := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelR()
	_, status_in_db, _, errR := pg_us_db.GetUserStatus(ctxR, userID)
	if errR != nil {
		sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 23", status, "")
		log.Printf("hDA: error getting root user status: %v", errR)
		return
	}

	if status_in_db == status && status == "root" {
		ctxCUS, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		errCUS := pg_us_db.ChangeUserStatus(ctxCUS, num_user_id, "user", "")
		if errCUS != nil {
			sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 23", status, "")
			log.Printf("hDA: error changing user status: %v", errCUS)
			return
		}
		sendSuccessMessage(bot, chatID, fmt.Sprintf("Админ: %s успешно удалён", login), status, "")
		intdACI, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			sendErrorMessage(bot, chatID, "Internal service error: попробуйте снова позже 27", status, "")
			return
		}

		sendSuccessMessage(bot, intdACI, "Вы были лишены полномочий администратора", "user", "")
		log.Printf("hDA: admin %s successfully removed", login)
		intUsID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			log.Printf("error whem parse parts[0]: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error 32", status, "")
			return
		}

		ctxDS, cancelDS := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelDS()
		errDS := pg_ses_db.DeleteSession(ctxDS, intUsID)
		if errDS != nil {
			log.Printf("error whem del session: %v", err)
			sendSuccessMessage(bot, chatID, "Internal service error", status, "")
			return
		}

		deleteSessionStatus(intUsID)
		log.Printf("hNA: admin sessiom deleted for: %d", intUsID)
		return
	} else {
		sendErrorMessage(bot, chatID, "Вам отказано в доступе", status, "")
		log.Printf("hDA: access denied for user %d", userID)
		return
	}
}
