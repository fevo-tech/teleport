/*
Copyright 2022 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package protocol

import "github.com/siddontang/go-mysql/mysql"

// StatementPreparePacket represents the COM_STMT_PREPARE command.
//
// https://dev.mysql.com/doc/internals/en/com-stmt-prepare.html
// https://mariadb.com/kb/en/com_stmt_prepare/
//
// COM_STMT_PREPARE creates a prepared statement from passed query string.
// Parameter placeholders are marked with "?" in the query. A COM_STMT_PREPARE
// response is expected from the server after sending this command.
type StatementPreparePacket struct {
	packet

	// query is the query to prepare.
	query string
}

// Query returns the query text.
func (p *StatementPreparePacket) Query() string {
	return p.query
}

// statementIDPacket represents a common packet format where statement ID is
// after the packet type.
//
// The statement ID is returned by the server in the COM_STMT_PREPARE response.
// All prepared statement packets except COM_STMT_PREPARE starts with the
// statement ID after the packet type to identify the prepared statement to
// use.
//
// The statement ID is an unsigned integer counter, usually starting at 1 for
// each client connection.
type statementIDPacket struct {
	packet

	// statementID is the ID of the associated statement.
	statementID uint32
}

// StatementID returns the statement ID.
func (p *statementIDPacket) StatementID() uint32 {
	return p.statementID
}

// StatementSendLongDataPacket represents the COM_STMT_SEND_LONG_DATA command.
//
// https://dev.mysql.com/doc/internals/en/com-stmt-send-long-data.html
// https://mariadb.com/kb/en/com_stmt_send_long_data/
//
// COM_STMT_SEND_LONG_DATA is used to send byte stream data to the server, and
// the server appends this data to the specified parameter upon receiving it.
// It is usually used for big blobs.
type StatementSendLongDataPacket struct {
	statementIDPacket

	// parameterID is the identifier of the parameter or column.
	parameterID uint16

	// data is the byte data sent in the packet.
	data []byte
}

// ParameterID returns the parameter ID.
func (p *StatementSendLongDataPacket) ParameterID() uint16 {
	return p.parameterID
}

// Data returns the data in bytes.
func (p *StatementSendLongDataPacket) Data() []byte {
	return p.data
}

// StatementExecutePacket represents the COM_STMT_EXECUTE command.
//
// https://dev.mysql.com/doc/internals/en/com-stmt-execute.html
// https://mariadb.com/kb/en/com_stmt_execute/
//
// COM_STMT_EXECUTE asks the server to execute a prepared statement, with the
// types and values for the placeholders.
//
// Statement ID "-1" (0xffffffff) can be used to indicate the last statement
// prepared on current connection, for MariaDB server version 10.2 and above.
type StatementExecutePacket struct {
	statementIDPacket

	// cursorFlag specifies type of the cursor.
	cursorFlag byte

	// iterations is the iteration count specified in the command. The MySQL
	// doc states that it is always 1.
	iterations uint32

	// nullBitmapAndParameters are raw packet bytes that represent a null
	// bitmap and parameters with types and values. They are not decoded in the
	// initial parsing because number of parameters is unknown.
	nullBitmapAndParameters []byte
}

// Parameters returns a slice of parameters.
func (p *StatementExecutePacket) Parameters(definitions []mysql.Field) (parameters []interface{}, ok bool) {
	// TODO(greedy52) implement parsing of null bitmap, parameter types, and
	// paramerter binary values.
	return nil, true
}

// StatementClosePacket represents the COM_STMT_CLOSE command.
//
// https://dev.mysql.com/doc/internals/en/com-stmt-close.html
// https://mariadb.com/kb/en/3-binary-protocol-prepared-statements-com_stmt_close/
//
// COM_STMT_CLOSE deallocates a prepared statement.
type StatementClosePacket struct {
	statementIDPacket
}

// StatementResetPacket represents the COM_STMT_RESET command.
//
// https://dev.mysql.com/doc/internals/en/com-stmt-reset.html
// https://mariadb.com/kb/en/com_stmt_reset/
//
// COM_STMT_RESET resets the data of a prepared statement which was accumulated
// with COM_STMT_SEND_LONG_DATA.
type StatementResetPacket struct {
	statementIDPacket
}

// StatementFetchPacket represents the COM_STMT_FETCH command.
//
// https://dev.mysql.com/doc/internals/en/com-stmt-fetch.html
// https://mariadb.com/kb/en/com_stmt_fetch/
//
// COM_STMT_FETCH fetch rows from a existing resultset after a
// COM_STMT_EXECUTE.
type StatementFetchPacket struct {
	statementIDPacket

	// rowsCount number of rows to fetch.
	rowsCount uint32
}

// RowsCount returns number of rows to fetch.
func (s *StatementFetchPacket) RowsCount() uint32 {
	return s.rowsCount
}

// StatementBulkExecutePacket represents the COM_STMT_BULK_EXECUTE command.
//
// https://mariadb.com/kb/en/com_stmt_bulk_execute/
//
// COM_STMT_BULK_EXECUTE executes a bulk insert of a previously prepared
// statement.
type StatementBulkExecutePacket struct {
	statementIDPacket

	// bulkFlag is a flag specifies either 64 (return generated auto-increment
	// IDs) or 128 (send types to server).
	bulkFlag uint16

	// parameters are raw packet bytes that contain parameter type and values.
	// They are not decoded in the initial parsing because number of parameters
	// is unknown.
	parameters []byte
}

// Parameters returns a slice of parameters.
func (p *StatementBulkExecutePacket) Parameters(definitions []mysql.Field) (parameters []interface{}, ok bool) {
	// TODO(greedy52) implement parsing of parameters from
	// COM_STMT_BULK_EXECUTE packet.
	return nil, true
}

// parseStatementPreparePacket parses packet bytes and returns a Packet if
// successful.
func parseStatementPreparePacket(rawPacket packet) (Packet, bool) {
	unread, ok := skipHeaderAndType(rawPacket.bytes)
	if !ok {
		return nil, false
	}

	return &StatementPreparePacket{
		packet: rawPacket,
		query:  string(unread),
	}, true
}

// parseStatementIDPacket parses packet bytes and returns a statementIDPacket
// if successful.
func parseStatementIDPacket(rawPacket packet) (statementIDPacket, []byte, bool) {
	unread, ok := skipHeaderAndType(rawPacket.bytes)
	if !ok {
		return statementIDPacket{}, nil, false
	}

	unread, statementID, ok := readUint32(unread)
	if !ok {
		return statementIDPacket{}, nil, false
	}

	return statementIDPacket{
		packet:      rawPacket,
		statementID: statementID,
	}, unread, true
}

// parseStatementSendLongDataPacket parses packet bytes and returns a Packet if
// successful.
func parseStatementSendLongDataPacket(rawPacket packet) (Packet, bool) {
	parent, unread, ok := parseStatementIDPacket(rawPacket)
	if !ok {
		return nil, false
	}

	unread, parameterID, ok := readUint16(unread)
	if !ok {
		return nil, false
	}

	return &StatementSendLongDataPacket{
		statementIDPacket: parent,
		parameterID:       parameterID,
		data:              unread,
	}, true
}

// parseStatementExecutePacket parses packet bytes and returns a Packet if
// successful.
func parseStatementExecutePacket(rawPacket packet) (Packet, bool) {
	parent, unread, ok := parseStatementIDPacket(rawPacket)
	if !ok {
		return nil, false
	}

	unread, cursorFlag, ok := readByte(unread)
	if !ok {
		return nil, false
	}

	unread, iterations, ok := readUint32(unread)
	if !ok {
		return nil, false
	}

	return &StatementExecutePacket{
		statementIDPacket:       parent,
		cursorFlag:              cursorFlag,
		iterations:              iterations,
		nullBitmapAndParameters: unread,
	}, true
}

// parseStatementClosePacket parses packet bytes and returns a Packet if
// successful.
func parseStatementClosePacket(rawPacket packet) (Packet, bool) {
	parent, _, ok := parseStatementIDPacket(rawPacket)
	if !ok {
		return nil, false
	}
	return &StatementClosePacket{
		statementIDPacket: parent,
	}, true
}

// parseStatementResetPacket parses packet bytes and returns a Packet if
// successful.
func parseStatementResetPacket(rawPacket packet) (Packet, bool) {
	parent, _, ok := parseStatementIDPacket(rawPacket)
	if !ok {
		return nil, false
	}
	return &StatementResetPacket{
		statementIDPacket: parent,
	}, true
}

// parseStatementFetchPacket parses packet bytes and returns a Packet if
// successful.
func parseStatementFetchPacket(rawPacket packet) (Packet, bool) {
	parent, unread, ok := parseStatementIDPacket(rawPacket)
	if !ok {
		return nil, false
	}

	_, rowsCount, ok := readUint32(unread)
	if !ok {
		return nil, false
	}
	return &StatementFetchPacket{
		statementIDPacket: parent,
		rowsCount:         rowsCount,
	}, true
}

// parseStatementBulkExecutePacket parses packet bytes and returns a Packet if
// successful.
func parseStatementBulkExecutePacket(rawPacket packet) (Packet, bool) {
	parent, unread, ok := parseStatementIDPacket(rawPacket)
	if !ok {
		return nil, false
	}

	unread, bulkFlag, ok := readUint16(unread)
	if !ok {
		return nil, false
	}

	return &StatementBulkExecutePacket{
		statementIDPacket: parent,
		bulkFlag:          bulkFlag,
		parameters:        unread,
	}, true
}
