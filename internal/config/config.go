package config

import (
	"fmt"
	"os"
	"strconv"
	//"strings"
	//env "github.com/joho/godotenv"
)

type Config struct {
	HEALTH_CHECK_PORT string
	DB_LIB_URL        string
	DB_US_URL         string
	BOT_TOKEN         string
	POSTGRES_LIB_SMOC int
	POSTGRES_LIB_SMIC int
	POSTGRES_US_SMOC  int
	POSTGRES_US_SMIC  int
	ADMIN_PASSWORD    string
	DOCX_ROOT_PATH    string
	DOCX_ADMIN_PATH   string
	DOCX_USER_PATH    string
	ROOT_ID           int64
}

func Loader() (*Config, error) {
	health_check_port := os.Getenv("HEALTH_CHECK_PORT")
	if health_check_port == "" {
		return nil, fmt.Errorf("error whem getting health check port")
	}

	bot_token := os.Getenv("BOT_TOKEN")
	if bot_token == "" {
		return nil, fmt.Errorf("error whem getting bot token")
	}

	db_lib_url := os.Getenv("DB_LIB_URL")
	if db_lib_url == "" {
		return nil, fmt.Errorf("error when getting db lib url")
	}
	db_lib_smoc := os.Getenv("POSTGRES_LIB_SMOC")
	if db_lib_smoc == "" {
		return nil, fmt.Errorf("error whem getting db lib smoc")
	}
	db_lib_smic := os.Getenv("POSTGRES_LIB_SMIC")
	if db_lib_smic == "" {
		return nil, fmt.Errorf("error whem getting db lib smic")
	}
	int_db_lib_smoc, err := strconv.Atoi(db_lib_smoc)
	if err != nil {
		return nil, err
	}
	int_db_lib_smic, err := strconv.Atoi(db_lib_smic)
	if err != nil {
		return nil, err
	}

	db_us_url := os.Getenv("DB_US_URL")
	if db_us_url == "" {
		return nil, fmt.Errorf("error whem getting db users url")
	}
	db_us_smoc := os.Getenv("POSTGRES_US_SMOC")
	if db_us_smoc == "" {
		return nil, fmt.Errorf("error whem getting db users smoc")
	}
	db_us_smic := os.Getenv("POSTGRES_US_SMIC")
	if db_us_smic == "" {
		return nil, fmt.Errorf("error whem getting db users smic")
	}
	int_db_us_smoc, err := strconv.Atoi(db_us_smoc)
	if err != nil {
		return nil, err
	}
	int_db_us_smic, err := strconv.Atoi(db_us_smic)
	if err != nil {
		return nil, err
	}

	root_id := os.Getenv("ROOT_ID")
	if root_id == "" {
		return nil, fmt.Errorf("error whem getting root id")
	}
	int_root_id, err := strconv.ParseInt(root_id, 10, 64)
	if err != nil {
		return nil, err
	}

	docx_root_puth := os.Getenv("DOCX_ROOT_PATH")
	if docx_root_puth == "" {
		return nil, fmt.Errorf("error whem getting docx root path")
	}
	docx_admin_path := os.Getenv("DOCX_ADMIN_PATH")
	if docx_admin_path == "" {
		return nil, fmt.Errorf("error whem getting docx admin path")
	}
	docx_user_path := os.Getenv("DOCX_USER_PATH")
	if docx_user_path == "" {
		return nil, fmt.Errorf("error whem getting docx user path")
	}

	return &Config{
		HEALTH_CHECK_PORT: health_check_port,
		DB_LIB_URL:        db_lib_url,
		DB_US_URL:         db_us_url,
		BOT_TOKEN:         bot_token,
		POSTGRES_LIB_SMOC: int_db_lib_smoc,
		POSTGRES_LIB_SMIC: int_db_lib_smic,
		POSTGRES_US_SMOC:  int_db_us_smoc,
		POSTGRES_US_SMIC:  int_db_us_smic,
		ROOT_ID:           int_root_id,
		DOCX_ROOT_PATH:    docx_root_puth,
		DOCX_ADMIN_PATH:   docx_admin_path,
		DOCX_USER_PATH:    docx_user_path,
	}, nil
}
