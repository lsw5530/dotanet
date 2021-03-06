package db

import (
	"database/sql"
	"dq/conf"
	"dq/log"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	Mydb *sql.DB
}

var DbOne *DB

func CreateDB() {
	DbOne = new(DB)
	DbOne.Init()
}

func (a *DB) Init() {

	ip := conf.Conf.DataBaseInfo["Ip"].(string)
	nameandpassword := conf.Conf.DataBaseInfo["NameAndPassword"].(string)
	databasename := conf.Conf.DataBaseInfo["DataBaseName"].(string)
	db, err := sql.Open("mysql", nameandpassword+"@"+ip+"/"+databasename)
	if err != nil {
		log.Error(err.Error())
	}
	err = db.Ping()
	if err != nil {
		log.Error(err.Error())
	}
	a.Mydb = db

	a.Mydb.SetMaxOpenConns(10000)
	a.Mydb.SetMaxIdleConns(500)
	a.Mydb.Ping()
}

func (a *DB) GetJSON(sqlString string) (string, error) {
	stmt, err := a.Mydb.Prepare(sqlString)
	if err != nil {
		return "", err
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		return "", err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	count := len(columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}
	jsonData, err := json.Marshal(tableData)
	if err != nil {
		return "", err
	}
	//log.Info(string(jsonData))
	return string(jsonData), nil
}

//创建快速新玩家
func (a *DB) CreateQuickPlayer(machineid string, platfom string, name string) int {

	id, _ := a.newUser(machineid, platfom, "", "", name)

	return id
}

//创建新玩家基础信息
func (a *DB) newUserBaseInfo(id int, name string) error {

	tx, _ := a.Mydb.Begin()

	res, err1 := tx.Exec("INSERT userbaseinfo (uid,name) values (?,?)",
		id, name)
	n, e := res.RowsAffected()
	if err1 != nil || n == 0 || e != nil {
		log.Info("INSERT userbaseinfo err")
		return tx.Rollback()
	}

	err1 = tx.Commit()
	return err1

}

//创建新玩家信息
func (a *DB) newUser(machineid string, platfom string, phonenumber string, openid string, name string) (int, error) {

	tx, _ := a.Mydb.Begin()

	res, err1 := tx.Exec("INSERT user (phonenumber,platform,machineid,wechat_id) values (?,?,?,?)",
		phonenumber, platfom, machineid, openid)
	n, e := res.RowsAffected()
	id, err2 := res.LastInsertId()
	if err1 != nil || n == 0 || e != nil || err2 != nil {
		log.Info("INSERT user err")
		return -1, tx.Rollback()
	}
	if name == "" {
		name = "yk_" + strconv.Itoa(int(id))
	}

	//day := time.Now().Format("2006-01-02")

	res, err1 = tx.Exec("INSERT userbaseinfo (uid,name) values (?,?)",
		id, name)
	//插入名字失败
	if err1 != nil {

		name = "yk_" + strconv.Itoa(int(id))
		res, err1 = tx.Exec("INSERT userbaseinfo (uid,name) values (?,?)",
			id, name)
		if err1 != nil {
			log.Info("INSERT userbaseinfo err")
			return -1, tx.Rollback()
		}
	}
	n, e = res.RowsAffected()
	if n == 0 || e != nil {
		log.Info("INSERT userbaseinfo err")
		return -1, tx.Rollback()
	}

	err1 = tx.Commit()
	if err1 == nil {
		return int(id), nil
	}
	return -1, err1

}

//检查快速登录
func (a *DB) CheckQuickLogin(machineid string, platfom string) int {
	var uid int

	stmt, err := a.Mydb.Prepare("SELECT uid FROM user where BINARY (machineid=? and platform=?)")

	if err != nil {
		log.Info(err.Error())
		return -1
	}
	defer stmt.Close()
	rows, err := stmt.Query(machineid, platfom)
	if err != nil {
		log.Info(err.Error())
		return uid
		//创建新账号
	}
	defer rows.Close()

	if rows.Next() {
		rows.Scan(&uid)
	} else {
		log.Info("no user:%s,%s", machineid, platfom)
	}

	return uid

}
func (a *DB) QueryAnything(sqlstr string, rowStruct interface{}) error {
	str, err := a.GetJSON(sqlstr)
	if err != nil {
		log.Info(err.Error())
		return err
	}
	//h2 := datamsg.MailInfo{}
	err = json.Unmarshal([]byte(str), rowStruct)
	if err != nil {
		log.Info(err.Error())
		return err
	}
	return nil
}

//func ()players := make([]db.DB_CharacterInfo, 0)

//获取玩家信息
func (a *DB) GetCharactersInfo(uid int32, playersInfo *[]DB_CharacterInfo) error {
	sqlstr := "SELECT * FROM characterinfo where uid=" + strconv.Itoa(int(uid))
	return a.QueryAnything(sqlstr, playersInfo)
}

//获取角色信息通过名字
func (a *DB) GetCharactersInfoByName(name string, playersInfo *[]DB_CharacterInfo) error {
	sqlstr := "SELECT * FROM characterinfo where name=" + "'" + name + "'"
	return a.QueryAnything(sqlstr, playersInfo)
}

//获取角色信息通过characterid
func (a *DB) GetCharactersInfoByCharacterid(characterid int32, playersInfo *[]DB_CharacterInfo) error {
	sqlstr := "SELECT * FROM characterinfo where characterid=" + strconv.Itoa(int(characterid))
	return a.QueryAnything(sqlstr, playersInfo)
}

//获取邮件信息信息通过多个邮件id
func (a *DB) GetMailsInfoByids(id []int, mailsInfo *[]DB_MailInfo) error {
	if len(id) <= 0 {
		return nil
	}
	sqlstr := "SELECT * FROM mail where"
	rulestr := ""
	for _, v := range id {
		if len(rulestr) > 0 {
			rulestr += " or"
		}
		rulestr = rulestr + " id=" + strconv.Itoa(int(v))
	}
	sqlstr += rulestr
	log.Info("sql:%s", sqlstr)
	return a.QueryAnything(sqlstr, mailsInfo)
}

//获取角色信息通过多个characterid
func (a *DB) GetCharactersInfoByCharacterids(characterid []int32, playersInfo *[]DB_CharacterInfo) error {
	if len(characterid) <= 0 {
		return nil
	}
	sqlstr := "SELECT * FROM characterinfo where"
	rulestr := ""
	for _, v := range characterid {
		if len(rulestr) > 0 {
			rulestr += " or"
		}
		rulestr = rulestr + " characterid=" + strconv.Itoa(int(v))
	}
	sqlstr += rulestr
	log.Info("sql:%s")
	return a.QueryAnything(sqlstr, playersInfo)
}

//获取交易所信息
func (a *DB) GetExchanges(commoditys *[]DB_PlayerItemTransactionInfo) error {

	sqlstr := "SELECT * FROM exchange"

	return a.QueryAnything(sqlstr, commoditys)
}

//获取公会信息
func (a *DB) GetGuilds(commoditys *[]DB_GuildInfo) error {

	sqlstr := "SELECT * FROM guild"

	return a.QueryAnything(sqlstr, commoditys)
}

//获取公会信息通过公会名字
func (a *DB) GetGuildsInfoByName(name string, guildsInfo *[]DB_GuildInfo) error {
	sqlstr := "SELECT * FROM guild where name=" + "'" + name + "'"
	return a.QueryAnything(sqlstr, guildsInfo)
}

//创建角色
func (a *DB) CreateCharacter(uid int32, name string, typeid int32) (error, int32) {

	//检查是否有重名的角色了
	players := make([]DB_CharacterInfo, 0)
	nameerr := a.GetCharactersInfoByName(name, &players)
	if nameerr != nil || len(players) > 0 {
		return errors.New("name repeat"), -1
	}

	tx, _ := a.Mydb.Begin()

	//a.Mydb.Exec()
	//sqlstr :=

	res, err1 := tx.Exec("INSERT characterinfo (uid,name,typeid) values (?,?,?)",
		uid, name, typeid)
	n, e := res.RowsAffected()
	characterid, err2 := res.LastInsertId()
	if err1 != nil || n == 0 || e != nil || err2 != nil {
		log.Info("INSERT characterinfo err")
		return tx.Rollback(), -1
	}

	err1 = tx.Commit()

	return err1, int32(characterid)
}

//添加邮件ID
func (a *DB) AddMail(mycharacterid int32, mailid int32) error {
	tx, e1 := a.Mydb.Begin()

	for tx == nil || e1 != nil {
		log.Info("AddMail :%s", e1.Error())
		time.Sleep(time.Millisecond * 2)
		tx, e1 = a.Mydb.Begin()
	}
	sqlstr := "UPDATE characterinfo SET mails= CONCAT(mails,';" + strconv.Itoa(int(mailid)) + "')"
	sqlstr += " where characterid=?"

	res, err1 := tx.Exec(sqlstr, mycharacterid)
	if err1 != nil {
		log.Info("err1 %s", err1.Error())
		return tx.Rollback()
	}
	n, e := res.RowsAffected()
	if n == 0 || e != nil {
		if e != nil {
			log.Info("AddMail err %s", e.Error())
		}

		return tx.Rollback()
	}

	err1 = tx.Commit()
	return err1
}

//添加好友请求
func (a *DB) AddFriendsRequest(mycharacterid int32, requestcid int32) error {
	tx, e1 := a.Mydb.Begin()

	for tx == nil || e1 != nil {
		log.Info("AddFriendsRequest :%s", e1.Error())
		time.Sleep(time.Millisecond * 2)
		tx, e1 = a.Mydb.Begin()
	}
	sqlstr := "UPDATE characterinfo SET friendsrequest= CONCAT(friendsrequest,';" + strconv.Itoa(int(requestcid)) + "')"
	sqlstr += " where characterid=?"

	res, err1 := tx.Exec(sqlstr, mycharacterid)
	if err1 != nil {
		log.Info("err1 %s", err1.Error())
		return tx.Rollback()
	}
	n, e := res.RowsAffected()
	if n == 0 || e != nil {
		if e != nil {
			log.Info("AddFriendsRequest err %s", e.Error())
		}

		return tx.Rollback()
	}

	err1 = tx.Commit()
	return err1
}

//添加好友
func (a *DB) AddFriends(mycharacterid int32, requestcid int32) error {
	tx, e1 := a.Mydb.Begin()

	for tx == nil || e1 != nil {
		log.Info("AddFriends :%s", e1.Error())
		time.Sleep(time.Millisecond * 2)
		tx, e1 = a.Mydb.Begin()
	}
	sqlstr := "UPDATE characterinfo SET friends= CONCAT(friends,';" + strconv.Itoa(int(requestcid)) + "')"
	sqlstr += " where characterid=?"

	res, err1 := tx.Exec(sqlstr, mycharacterid)
	if err1 != nil {
		log.Info("err1 %s", err1.Error())
		return tx.Rollback()
	}
	n, e := res.RowsAffected()
	if n == 0 || e != nil {
		if e != nil {
			log.Info("AddFriends err %s", e.Error())
		}

		return tx.Rollback()
	}

	err1 = tx.Commit()
	return err1
}

//保存角色信息
func (a *DB) SaveCharacter(playerInfo DB_CharacterInfo) error {
	tx, e1 := a.Mydb.Begin()

	for tx == nil || e1 != nil {
		log.Info("SaveCharacter11 :%s", e1.Error())
		time.Sleep(time.Millisecond * 2)
		tx, e1 = a.Mydb.Begin()

	}

	//要存的数据
	datastring := make(map[string]interface{})
	datastring["name"] = playerInfo.Name
	datastring["level"] = playerInfo.Level
	datastring["experience"] = playerInfo.Experience
	datastring["gold"] = playerInfo.Gold
	datastring["diamond"] = playerInfo.Diamond
	datastring["hp"] = playerInfo.HP
	datastring["mp"] = playerInfo.MP
	datastring["sceneid"] = playerInfo.SceneID
	datastring["scenename"] = playerInfo.SceneName
	datastring["x"] = playerInfo.X
	datastring["y"] = playerInfo.Y
	datastring["skill"] = playerInfo.Skill
	datastring["item1"] = playerInfo.Item1
	datastring["item2"] = playerInfo.Item2
	datastring["item3"] = playerInfo.Item3
	datastring["item4"] = playerInfo.Item4
	datastring["item5"] = playerInfo.Item5
	datastring["item6"] = playerInfo.Item6
	datastring["baginfo"] = playerInfo.BagInfo
	datastring["itemskillcd"] = playerInfo.ItemSkillCDInfo
	datastring["remainexperience"] = playerInfo.RemainExperience
	datastring["getexperienceday"] = playerInfo.GetExperienceDay
	datastring["remainerevivetime"] = playerInfo.RemainReviveTime
	datastring["killcount"] = playerInfo.KillCount
	datastring["continuitykillcount"] = playerInfo.ContinuityKillCount
	datastring["diecount"] = playerInfo.DieCount
	datastring["killgetgold"] = playerInfo.KillGetGold
	datastring["friends"] = playerInfo.Friends
	datastring["friendsrequest"] = playerInfo.FriendsRequest
	datastring["watchvediocountoneday"] = playerInfo.WatchVedioCountOneDay
	datastring["mails"] = playerInfo.Mails
	datastring["guildid"] = playerInfo.GuildId
	datastring["guildpinlevel"] = playerInfo.GuildPinLevel
	datastring["guildpinexperience"] = playerInfo.GuildPinExperience
	datastring["guildpost"] = playerInfo.GuildPost

	sqlstr := "UPDATE characterinfo SET "
	count := 0
	for k, v := range datastring {

		switch v.(type) {

		case string:
			sqlstr += k + "=" + "'" + v.(string) + "'"
			break
		case int:
			sqlstr += k + "=" + strconv.Itoa(v.(int))
			break
		case int32:
			sqlstr += k + "=" + strconv.Itoa(int(v.(int32)))
			break
		case int64:
			sqlstr += k + "=" + strconv.Itoa(int(v.(int64)))
			break
		case float64:
			sqlstr += k + "=" + strconv.FormatFloat(float64(v.(float64)), 'f', 4, 32)
			break
		case float32:
			sqlstr += k + "=" + strconv.FormatFloat(float64(v.(float32)), 'f', 4, 32)
			break
		}
		if count == len(datastring)-1 {

		} else {
			sqlstr += ","
		}
		count++

	}
	sqlstr += " where characterid=?"

	//log.Info("SaveCharacter:%s ---%d", sqlstr, playerInfo.Characterid)

	res, err1 := tx.Exec(sqlstr, playerInfo.Characterid)
	if err1 != nil {
		log.Info("err1 %s", err1.Error())
		return tx.Rollback()
	}
	n, e := res.RowsAffected()
	if n == 0 || e != nil {
		if e != nil {
			log.Info("SaveCharacter err %s", e.Error())
		}

		return tx.Rollback()
	}

	err1 = tx.Commit()
	return err1
}

//创建公会
func (a *DB) CreateGuild(name string, day string) (error, int32) {
	tx, _ := a.Mydb.Begin()

	res, err1 := tx.Exec("INSERT guild (name,createday) values (?,?)", name, day)
	n, e := res.RowsAffected()
	id, err2 := res.LastInsertId()
	if err1 != nil || n == 0 || e != nil || err2 != nil {
		log.Info("CreateGuild  err")
		return tx.Rollback(), -1
	}

	err1 = tx.Commit()

	return err1, int32(id)
}

//保存公会信息
func (a *DB) SaveGuild(guild DB_GuildInfo) error {
	tx, e1 := a.Mydb.Begin()

	for tx == nil || e1 != nil {
		log.Info("SaveGuild :%s", e1.Error())
		time.Sleep(time.Millisecond * 2)
		tx, e1 = a.Mydb.Begin()

	}

	//要存的数据
	datastring := make(map[string]interface{})
	datastring["id"] = guild.Id
	datastring["presidentCharacterid"] = guild.PresidentCharacterid
	datastring["level"] = guild.Level
	datastring["experience"] = guild.Experience
	datastring["notice"] = guild.Notice
	datastring["joinaudit"] = guild.Joinaudit
	datastring["joinlevellimit"] = guild.Joinlevellimit
	datastring["characters"] = guild.Characters
	datastring["requestjoincharacters"] = guild.RequestJoinCharacters

	sqlstr := "UPDATE guild SET "
	count := 0
	for k, v := range datastring {

		switch v.(type) {

		case string:
			sqlstr += k + "=" + "'" + v.(string) + "'"
			break
		case int:
			sqlstr += k + "=" + strconv.Itoa(v.(int))
			break
		case int32:
			sqlstr += k + "=" + strconv.Itoa(int(v.(int32)))
			break
		case int64:
			sqlstr += k + "=" + strconv.Itoa(int(v.(int64)))
			break
		case float64:
			sqlstr += k + "=" + strconv.FormatFloat(float64(v.(float64)), 'f', 4, 32)
			break
		case float32:
			sqlstr += k + "=" + strconv.FormatFloat(float64(v.(float32)), 'f', 4, 32)
			break
		}
		if count == len(datastring)-1 {

		} else {
			sqlstr += ","
		}
		count++

	}
	sqlstr += " where id=?"

	//log.Info("SaveCharacter:%s ---%d", sqlstr, playerInfo.Characterid)

	res, err1 := tx.Exec(sqlstr, guild.Id)
	if err1 != nil {
		log.Info("err1 %s", err1.Error())
		return tx.Rollback()
	}
	n, e := res.RowsAffected()
	if n == 0 || e != nil {
		if e != nil {
			log.Info("guild err %s", e.Error())
		}

		return tx.Rollback()
	}

	err1 = tx.Commit()
	return err1
}

//创建并保存上架到交易所的道具
func (a *DB) CreateAndSaveCommodity(mailInfo *DB_PlayerItemTransactionInfo) {
	_, id := a.CreateCommodity()
	if id < 0 {
		return
	}

	mailInfo.Id = id

	a.SaveCommodity(*mailInfo)
}

//创建商品
func (a *DB) CreateCommodity() (error, int32) {
	tx, _ := a.Mydb.Begin()
	res, err1 := tx.Exec("INSERT exchange (level) values (?)", 1)
	n, e := res.RowsAffected()
	characterid, err2 := res.LastInsertId()
	if err1 != nil || n == 0 || e != nil || err2 != nil {
		log.Info("INSERT mail err")
		return tx.Rollback(), -1
	}

	err1 = tx.Commit()

	return err1, int32(characterid)
}

//删除商品
func (a *DB) DeleteCommodity(id int32) error {
	tx, _ := a.Mydb.Begin()
	res, err1 := tx.Exec("DELETE FROM exchange WHERE id=" + strconv.Itoa(int(id)))
	n, e := res.RowsAffected()
	_, err2 := res.LastInsertId()
	if err1 != nil || n == 0 || e != nil || err2 != nil {
		log.Info("DeleteCommodity err")
		return tx.Rollback()
	}

	err1 = tx.Commit()

	return err1
}

//保存商品信息
func (a *DB) SaveCommodity(mailInfo DB_PlayerItemTransactionInfo) error {
	tx, e1 := a.Mydb.Begin()

	for tx == nil || e1 != nil {
		log.Info("SaveCommodity :%s", e1.Error())
		time.Sleep(time.Millisecond * 2)
		tx, e1 = a.Mydb.Begin()

	}

	//要存的数据
	datastring := make(map[string]interface{})
	datastring["id"] = mailInfo.Id
	datastring["itemid"] = mailInfo.ItemID
	datastring["level"] = mailInfo.Level
	datastring["pricetype"] = mailInfo.PriceType
	datastring["price"] = mailInfo.Price
	datastring["sellerUid"] = mailInfo.SellerUid
	datastring["sellerCharacterid"] = mailInfo.SellerCharacterid
	datastring["shelftime"] = mailInfo.ShelfTime

	sqlstr := "UPDATE exchange SET "
	count := 0
	for k, v := range datastring {

		switch v.(type) {

		case string:
			sqlstr += k + "=" + "'" + v.(string) + "'"
			break
		case int:
			sqlstr += k + "=" + strconv.Itoa(v.(int))
			break
		case int32:
			sqlstr += k + "=" + strconv.Itoa(int(v.(int32)))
			break
		case int64:
			sqlstr += k + "=" + strconv.Itoa(int(v.(int64)))
			break
		case float64:
			sqlstr += k + "=" + strconv.FormatFloat(float64(v.(float64)), 'f', 4, 32)
			break
		case float32:
			sqlstr += k + "=" + strconv.FormatFloat(float64(v.(float32)), 'f', 4, 32)
			break
		}
		if count == len(datastring)-1 {

		} else {
			sqlstr += ","
		}
		count++

	}
	sqlstr += " where id=?"

	//log.Info("SaveCharacter:%s ---%d", sqlstr, playerInfo.Characterid)

	res, err1 := tx.Exec(sqlstr, mailInfo.Id)
	if err1 != nil {
		log.Info("err1 %s", err1.Error())
		return tx.Rollback()
	}
	n, e := res.RowsAffected()
	if n == 0 || e != nil {
		if e != nil {
			log.Info("mail err %s", e.Error())
		}

		return tx.Rollback()
	}

	err1 = tx.Commit()
	return err1
}

//创建并保存邮件
func (a *DB) CreateAndSaveMail(mailInfo *DB_MailInfo) {
	_, id := a.CreateMail()
	if id < 0 {
		return
	}

	mailInfo.Id = id

	a.SaveMail(*mailInfo)
}

//创建邮件
func (a *DB) CreateMail() (error, int32) {
	tx, _ := a.Mydb.Begin()

	//a.Mydb.Exec()
	//sqlstr :=
	day := time.Now().Format("2006-01-02")

	res, err1 := tx.Exec("INSERT mail (date) values (?)", day)
	n, e := res.RowsAffected()
	characterid, err2 := res.LastInsertId()
	if err1 != nil || n == 0 || e != nil || err2 != nil {
		log.Info("INSERT mail err")
		return tx.Rollback(), -1
	}

	err1 = tx.Commit()

	return err1, int32(characterid)
}

//保存邮件信息
func (a *DB) SaveMail(mailInfo DB_MailInfo) error {
	tx, e1 := a.Mydb.Begin()

	for tx == nil || e1 != nil {
		log.Info("SaveMail :%s", e1.Error())
		time.Sleep(time.Millisecond * 2)
		tx, e1 = a.Mydb.Begin()

	}

	//要存的数据
	datastring := make(map[string]interface{})
	datastring["id"] = mailInfo.Id
	datastring["sendname"] = mailInfo.Sendname
	datastring["title"] = mailInfo.Title
	datastring["content"] = mailInfo.Content
	datastring["recUid"] = mailInfo.RecUid
	datastring["recCharacterid"] = mailInfo.RecCharacterid
	//datastring["date"] = mailInfo.Date
	datastring["rewardstr"] = mailInfo.Rewardstr
	datastring["getstate"] = mailInfo.Getstate

	sqlstr := "UPDATE mail SET "
	count := 0
	for k, v := range datastring {

		switch v.(type) {

		case string:
			sqlstr += k + "=" + "'" + v.(string) + "'"
			break
		case int:
			sqlstr += k + "=" + strconv.Itoa(v.(int))
			break
		case int32:
			sqlstr += k + "=" + strconv.Itoa(int(v.(int32)))
			break
		case int64:
			sqlstr += k + "=" + strconv.Itoa(int(v.(int64)))
			break
		case float64:
			sqlstr += k + "=" + strconv.FormatFloat(float64(v.(float64)), 'f', 4, 32)
			break
		case float32:
			sqlstr += k + "=" + strconv.FormatFloat(float64(v.(float32)), 'f', 4, 32)
			break
		}
		if count == len(datastring)-1 {

		} else {
			sqlstr += ","
		}
		count++

	}
	sqlstr += " where id=?"

	//log.Info("SaveCharacter:%s ---%d", sqlstr, playerInfo.Characterid)

	res, err1 := tx.Exec(sqlstr, mailInfo.Id)
	if err1 != nil {
		log.Info("err1 %s", err1.Error())
		return tx.Rollback()
	}
	n, e := res.RowsAffected()
	if n == 0 || e != nil {
		if e != nil {
			log.Info("mail err %s", e.Error())
		}

		return tx.Rollback()
	}

	err1 = tx.Commit()
	return err1
}

func (a *DB) test() {

}

func (a *DB) Close() {
	a.Mydb.Close()
}
