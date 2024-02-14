package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
)

type EVChargingChaincode struct{}

type ChargingSession struct {
	SessionID string  `json:"sessionID"`
	VehicleID string  `json:"vehicleID"`
	StartTime string  `json:"startTime"`
	EndTime   string  `json:"endTime"`
	Energy    float64 `json:"energy"`
}

func (cc *EVChargingChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (cc *EVChargingChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()

	switch fn {
	case "registerSession":
		return cc.registerSession(stub, args)
	case "querySession":
		return cc.querySession(stub, args)
	case "updateSession":
		return cc.updateSession(stub, args)
	case "deleteSession":
		return cc.deleteSession(stub, args)
	default:
		return shim.Error("Available functions: registerSession, querySession, updateSession, deleteSession")
	}
}

func (cc *EVChargingChaincode) isSubmitterAuthorized(stub shim.ChaincodeStubInterface, authorizedMspID string) bool {
	creator, err := stub.GetCreator()
	if err != nil {
		fmt.Printf("Error getting transaction submitter: %s\n", err)
		return false
	}

	sId := &msp.SerializedIdentity{}
	if err := proto.Unmarshal(creator, sId); err != nil {
		fmt.Printf("Could not unmarshal SerializedIdentity: %s\n", err)
		return false
	}

	return sId.Mspid == authorizedMspID
}

func (cc *EVChargingChaincode) registerSession(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if !cc.isSubmitterAuthorized(stub, "AuthorizedMspID") { //TODO: Replace with actual MSP ID
		return shim.Error("Unauthorized to register a charging session")
	}

	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting 5")
	}

	sessionID := args[0]
	if sessionBytes, _ := stub.GetState(sessionID); sessionBytes != nil {
		return shim.Error("SessionID already exists")
	}

	energy, err := strconv.ParseFloat(args[4], 64)
	if err != nil || energy <= 0 {
		return shim.Error("Energy consumption must be a positive numeric value")
	}

	session := ChargingSession{
		SessionID: args[0],
		VehicleID: args[1],
		StartTime: args[2],
		EndTime:   args[3],
		Energy:    energy,
	}

	sessionBytes, err := json.Marshal(session)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to encode session data: %s", err))
	}

	err = stub.PutState(sessionID, sessionBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to register charging session: %s", err))
	}

	stub.SetEvent("evSessionRegistered", []byte(sessionID))
	return shim.Success(nil)
}

func (cc *EVChargingChaincode) querySession(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	sessionBytes, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get charging session: %s", err))
	}
	if sessionBytes == nil {
		return shim.Error("Session not found")
	}

	return shim.Success(sessionBytes)
}

func (cc *EVChargingChaincode) updateSession(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// Similar validation as in registerSession
	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting 5")
	}

	sessionID := args[0]
	sessionBytes, err := stub.GetState(sessionID)
	if err != nil || sessionBytes == nil {
		return shim.Error("Session not found")
	}

	energy, err := strconv.ParseFloat(args[4], 64)
	if err != nil || energy <= 0 {
		return shim.Error("Invalid energy value")
	}

	// Assuming all arguments are valid, create a new session object
	session := ChargingSession{
		SessionID: sessionID,
		VehicleID: args[1],
		StartTime: args[2],
		EndTime:   args[3],
		Energy:    energy,
	}

	sessionBytes, err = json.Marshal(session)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error marshaling session: %s", err))
	}

	err = stub.PutState(sessionID, sessionBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to update session: %s", err))
	}

	return shim.Success(nil)
}

func (cc *EVChargingChaincode) deleteSession(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	err := stub.DelState(args[0])
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to delete session: %s", err))
	}

	return shim.Success(nil)
}

func (cc *EVChargingChaincode) querySessionsWithPagination(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) < 3 {
		return shim.Error("Incorrect number of arguments. Expecting at least 3")
	}

	startKey, endKey := args[0], args[1]
	pageSize, err := strconv.ParseInt(args[2], 10, 32)
	if err != nil {
		return shim.Error("PageSize argument must be a numeric string")
	}
	bookmark := ""
	if len(args) > 3 {
		bookmark = args[3]
	}

	resultsIterator, responseMetadata, err := stub.GetStateByRangeWithPagination(startKey, endKey, int32(pageSize), bookmark)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	sessions := []ChargingSession{}
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		session := ChargingSession{}
		json.Unmarshal(queryResponse.Value, &session)
		sessions = append(sessions, session)
	}

	sessionsJSON, err := json.Marshal(sessions)
	if err != nil {
		return shim.Error(err.Error())
	}

	metadataJSON, err := json.Marshal(responseMetadata)
	if err != nil {
		return shim.Error(err.Error())
	}

	finalJSON := fmt.Sprintf("{\"Sessions\":%s,\"Metadata\":%s}", string(sessionsJSON), string(metadataJSON))
	return shim.Success([]byte(finalJSON))
}
func (cc *EVChargingChaincode) querySessionsByVehicle(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	queryString := fmt.Sprintf("{\"selector\":{\"vehicleID\":\"%s\"}}", args[0])

	queryResults, err := stub.GetQueryResult(queryString)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get query results: %s", err))
	}
	defer queryResults.Close()

	sessions := []ChargingSession{}
	for queryResults.HasNext() {
		queryResponse, err := queryResults.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		session := ChargingSession{}
		json.Unmarshal(queryResponse.Value, &session)
		sessions = append(sessions, session)
	}

	sessionsJSON, err := json.Marshal(sessions)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(sessionsJSON)
}
