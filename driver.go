package scaledriver

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// ScaleData представляет данные, полученные от весов
type ScaleData struct {
    Weight   int32
    Division byte
    Stable   bool
    Net      bool
    Zero     bool
    Tare     int32
}

// ScaleDriver интерфейс для драйвера весов
type ScaleDriver interface {
    OpenConnection() error
    ReadWeight() (*ScaleData, error)
    CloseConnection() error
}

// TCPScaleDriver реализует ScaleDriver для TCP соединения
type TCPScaleDriver struct {
    conn    net.Conn
    address string
}

// NewTCPScaleDriver создает новый экземпляр TCPScaleDriver
func NewTCPScaleDriver(address string) *TCPScaleDriver {
    return &TCPScaleDriver{address: address}
}

// OpenConnection устанавливает TCP соединение с весами
func (d *TCPScaleDriver) OpenConnection() error {
    conn, err := net.DialTimeout("tcp", d.address, 5*time.Second)
    if err != nil {
        return fmt.Errorf("ошибка подключения к весам: %v", err)
    }
    d.conn = conn
    return nil
}

// CloseConnection закрывает соединение с весами
func (d *TCPScaleDriver) CloseConnection() error {
    if d.conn != nil {
        return d.conn.Close()
    }
    return nil
}

// ReadWeight считывает данные с весов
func (d *TCPScaleDriver) ReadWeight() (*ScaleData, error) {
    // Отправляем команду CMD_GET_MASSA
    if err := d.sendCommand(0x23); err != nil {
        return nil, err
    }

    // Читаем ответ
    resp, err := d.readResponse()
    if err != nil {
        return nil, err
    }

    // Парсим ответ
    if len(resp) < 13 {
        return nil, fmt.Errorf("некорректная длина ответа")
    }

    data := &ScaleData{
        Weight:   int32(binary.LittleEndian.Uint32(resp[1:5])),
        Division: resp[5],
        Stable:   resp[6] == 1,
        Net:      resp[7] == 1,
        Zero:     resp[8] == 1,
    }

    // Проверяем, есть ли данные о таре
    if len(resp) >= 17 {
        data.Tare = int32(binary.LittleEndian.Uint32(resp[9:13]))
    }

    return data, nil
}

// sendCommand отправляет команду весам
func (d *TCPScaleDriver) sendCommand(cmd byte) error {
    msg := []byte{0xF8, 0x55, 0xCE, 0x01, 0x00, cmd}
    crc := calculateCRC(msg[5:])
    msg = append(msg, byte(crc), byte(crc>>8))

    _, err := d.conn.Write(msg)
    return err
}

// readResponse читает ответ от весов
func (d *TCPScaleDriver) readResponse() ([]byte, error) {
    d.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
    resp := make([]byte, 1024)
    n, err := d.conn.Read(resp)
    if err != nil {
        return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
    }
    return resp[:n], nil
}

// calculateCRC вычисляет CRC-16-CCITT
func calculateCRC(data []byte) uint16 {
    crc := uint16(0xFFFF)
    for _, b := range data {
        crc ^= uint16(b) << 8
        for i := 0; i < 8; i++ {
            if crc&0x8000 != 0 {
                crc = (crc << 1) ^ 0x1021
            } else {
                crc <<= 1
            }
        }
    }
    return crc
}
