using Neo.SmartContract.Framework;
using Neo.SmartContract.Framework.Services.Neo;
using Helper = Neo.SmartContract.Framework.Helper;
using System;
using System.ComponentModel;
using System.Numerics;

[assembly: ContractTitle("Archon Cloud Storage")]
[assembly: ContractDescription("Contract for storage using blockchain storage providers")]
[assembly: ContractVersion("1.9")]
[assembly: ContractAuthor("Archon Inc")]
[assembly: ContractEmail("support@archon.cloud")]
[assembly: Features(ContractPropertyState.HasStorage | ContractPropertyState.HasDynamicInvoke | ContractPropertyState.Payable)]

namespace Neo.SmartContract
{
public class CloudStorageContract : Framework.SmartContract
{
     // Note: do not use "Reverse()". From Neo team: .Reverse() worked in older version compilers, and in V2.6.0 it's better not to use it.
    private static readonly byte[] cgasContractHash = "76db3192722022eb7841038246dc8fa636dcf274".HexToBytes(); // testnet, reversed
    //private static readonly byte[] cgasContractHash = "f3c7a1170d2e9cb33827210daedf257db0c38c2a".HexToBytes(); //private, reversed
    private static readonly string spProfilesMap = "addressToProfile";
    private static readonly string nodeIdToAddrMap = "nodeIdToAddr";
    private static readonly string addressToUserNameMap = "addressToUserName";

    // Used to call another contract
    public delegate object DynCall(string method, object[] args);

    [DisplayName("notification")]
    public static event deleNotification Notification;
    public delegate void deleNotification(object notification);

    [DisplayName("error")]
    public static event deleError Error;
    public delegate void deleError(string message);
    public static object Main(string method, object[] args)
    {
        if (Runtime.Trigger == TriggerType.Verification)
        {
            /*A verification trigger is used to invoke the contract as a verification function, accepting multiple parameters and returning a valid
                * Boolean value, indicating the validity of the transaction or block.
                * The contract code is executed to verify whether a transaction involving assets owned by the
                * contract address should be allowed to succeed.
            */
            return true;
        }
        if (Runtime.Trigger == TriggerType.Application)
        {
            // activated by InvocationTransaction
            // the transaction is recorded in the blockchain irrespective of whether the smart contract execution succeeds or fails
            if (method=="getUserName")
            {
                return GetUserName((byte[])args[0]);
            }
            if (method=="registerUserName")
            {
                return RegisterUserName((byte[])args[0], (string)args[1]);
            }
            if (method=="unregisterUserName")
            {
                return UnregisterUserName((byte[])args[0]);
            }
            if (method == "registerStorageProvider")
            {
                return RegisterStorageProvider((byte[])args[0], (string)args[1], (string)args[2]);
            }
            if (method == "unregisterStorageProvider")
            {
                return UnregisterStorageProvider((byte[])args[0], (string)args[1]);
            }
            if (method == "getStorageProviderProfile")
            {
                return GetStorageProviderProfile((byte[])args[0]);
            }
            if (method == "getStorageProviderMinAsk")
            {
                return GetStorageProviderMinAsk((byte[])args[0]);
            }
            if (method == "getStorageProviderAddress")
            {
                return GetStorageProviderAddress((string)args[0]);
            }
            if (method == "proposeUpload")
            {
                return ProposeUpload((byte[])args[0], (byte[])args[1], (string)args[2], (string)args[3], (byte[])args[4]);
            }
            if (method == "version")
            {
                return Version();
            }
            if (method == "getCgasName")
            {
                return GetCgasName();
            }
            if (method == "transferCGAS")
            {
                return TransferCGAS((byte[])args[0], (byte[])args[1], (BigInteger)args[2]);
            }
            if (method == "mintCGAS")
            {
                return MintCGAS();
            }
            errorMessage("Unknown method:"+method);
        }
        return true;
    }

    [DisplayName("transferCGAS")]
    public static bool TransferCGAS(byte[] from, byte[] to, BigInteger amount)
    {
        var cgasContract = (DynCall)cgasContractHash.ToDelegate();
        var success = (bool)cgasContract("transfer", new object[] { from, to, amount });
        return success;
    }

    [DisplayName("mintCGAS")]
    public static bool MintCGAS()
    {
        var cgasContract = (DynCall)cgasContractHash.ToDelegate();
        var success = (bool)cgasContract("mintTokens", null);
        return success;
    }

    private static bool IsPayable(byte[] to)
    {
        var c = Blockchain.GetContract(to); //0.1
        return c == null || c.IsPayable;
    }

    [DisplayName("version")]
    public static string Version()
    {
        return "Archon Cloud Storage V1.8";
    }

    [DisplayName("getCgasName")]
    public static string GetCgasName()
    {
        var cgasContract = (DynCall) cgasContractHash.ToDelegate();
        var cgasName = (string) cgasContract("name", null );
        return cgasName;
    }

     [DisplayName("proposeUpload")]
     public static int ProposeUpload(byte[] uploaderAddress, byte[] spAddress, string paymentS, string mBytesS, byte[] uploadInfo)
     {
         // Returns 0 on success
        if (uploaderAddress.Length != 20)
        {
            // Not registered
            errorMessage("Invalid uploader address");
            return 7;
        }
        if (spAddress.Length != 20)
        {
            // Not registered
            errorMessage("Invalid SP address");
            return 7;
        }

        if (!Runtime.CheckWitness(uploaderAddress))
        {
            errorMessage("Transaction not signed by uploader");
            return 1;
        }
        var payment = convertToInt(paymentS);
        if (payment < 0)
        {
            errorMessage("Payment is negative");
            return 21;
        }
        var mBytes = convertToInt(mBytesS);
        if (mBytes <= 0)
        {
            // For debugging
            errorMessage("MBytes must be positive");
            return 22;
        }
        StorageMap addressToUserName = Storage.CurrentContext.CreateMap(addressToUserNameMap);
        var name = addressToUserName.Get(uploaderAddress);
        if (name == null)
        {
            errorMessage("Uploader not registered");
            return 3;
        }
        var cgas = (DynCall) cgasContractHash.ToDelegate();
        var uploaderCgasBalance = (BigInteger) cgas("balanceOf", new object[] { uploaderAddress });
        if (uploaderCgasBalance < payment)
        {
            errorMessage("Uploader CGAS balance is too low");
            return 6;
        }

        // Verify that all min asks are satisfied
        // minAsk in Gas per MByte
        if (!IsPayable(spAddress))
        {
            errorMessage("SP not payable");
            return 10;
        }
        var profile = GetStorageProviderProfile(spAddress);
        if (profile == "")
        {
            errorMessage("SP not registered");
            return 11;
        }
        var minAsk = getStorageProviderMinAskFromProfile(profile);
        if (minAsk > payment*mBytes)
        {
            errorMessage("SP min payment not satisfied");
            return 5;
        }
        if (payment > 0)
        {
            // Pay in CGAS
            // Transfer from the uploader to the SP
            bool success = TransferCGAS(uploaderAddress,spAddress,payment);
            if (!success)
            {
                // TODO: revert transfers done up to this point
                errorMessage("CGAS transfer failed");
                return 9;
            }
        }
        // Success. Record transfer info. Will be retrieved later by the SP
        Notification(uploadInfo);
        return 0;
    }

    [DisplayName("registerStorageProvider")]
    public static object RegisterStorageProvider(byte[] address, string nodeId, string profile)
    {
        if (!(Runtime.CheckWitness(address)))
        {
            // Must be invoked by SP
            errorMessage("Address mismatch");
            return null;
        }

        StorageMap addressToProfile = Storage.CurrentContext.CreateMap(spProfilesMap);
        var prof = addressToProfile.Get(address);
        if (prof != null) {
            // Already registered, must unregister first
            errorMessage("Already registered");
            return null;
        }

        StorageMap nodeIdToAddr = Storage.CurrentContext.CreateMap(nodeIdToAddrMap);
        var existingAddr = nodeIdToAddr.Get(nodeId);
        if (existingAddr != null)
        {
            errorMessage("Node id already exists");
            return null;
        }
        
        addressToProfile.Put(address, profile);
        nodeIdToAddr.Put(nodeId, address);
        return null;
    }

    [DisplayName("unregisterStorageProvider")]
    public static object UnregisterStorageProvider(byte[] address, string nodeId)
    {
        if (!(Runtime.CheckWitness(address)))
        {
            // Must be invoked by the SP that is unregistering
            errorMessage("Not owner of address");
            return null;
        }

        StorageMap addressToProfile = Storage.CurrentContext.CreateMap(spProfilesMap);
        addressToProfile.Delete(address);
        StorageMap nodeIdToAddr = Storage.CurrentContext.CreateMap(nodeIdToAddrMap);
        nodeIdToAddr.Delete(nodeId);
        return null;
    }

    [DisplayName("getStorageProviderProfile")]
    public static string GetStorageProviderProfile(byte[] address)
    {
        StorageMap addressToProfile = Storage.CurrentContext.CreateMap(spProfilesMap);
        var val = addressToProfile.Get(address);
        if (val == null)
            return "";

        return val.AsString();
    }

    [DisplayName("getStorageProviderMinAsk")]
    public static int GetStorageProviderMinAsk(byte[] address)
    {
        // e.g.: "200|100|USA|QmY9EWeuE4yL4ccvwLtX9PP8baWcWE4Kw4fkGh7ZhsTvkc"
        // This is mostly used for debugging
        string profile = GetStorageProviderProfile(address);
        int i = getStorageProviderMinAskFromProfile(profile);
        return i;
    }
    private static int getStorageProviderMinAskFromProfile(string profile)
    {
        // e.g.: "200|100|USA|QmY9EWeuE4yL4ccvwLtX9PP8baWcWE4Kw4fkGh7ZhsTvkc"
        // minAsk if first (200 above)
        string s = extractString(profile,0);
        int i = convertToInt(s);
        return i;
    }

    [DisplayName("getStorageProviderAddress")]
    public static string GetStorageProviderAddress(string nodeId)
    {
        StorageMap nodeToAddr = Storage.CurrentContext.CreateMap(nodeIdToAddrMap);
        var val = nodeToAddr.Get(nodeId);
        if (val == null)
            return null;

        return val.AsString();
    }

    [DisplayName("getUserName")]
    public static string GetUserName(byte[] address)
    {
        StorageMap addressToUserName = Storage.CurrentContext.CreateMap(addressToUserNameMap);
        var name = addressToUserName.Get(address);
        if (name == null)
        {
            return "";
        }
        return name.AsString();
    }

    [DisplayName("registerUserName")]
    public static int RegisterUserName(byte[] address, string userName)
    {
        if (!Runtime.CheckWitness(address))
        {
            // Only owner can register
            errorMessage("Invalid address");
            return 1;
        }

        StorageMap addressToUserName = Storage.CurrentContext.CreateMap(addressToUserNameMap);
        var name = addressToUserName.Get(address);
        if (name != null)
        {
            if (name.AsString() == userName)
                return 0;
            errorMessage("Already registered");
            return 2;
        }
        addressToUserName.Put(address, userName);
        return 0;
    }

    [DisplayName("unregisterUserName")]
    public static int UnregisterUserName(byte[] address)
    {
        if (!Runtime.CheckWitness(address))
            // Only owner can unregister
            return 1;

        StorageMap addressToUserName = Storage.CurrentContext.CreateMap(addressToUserNameMap);
        addressToUserName.Delete(address);
        return 0;
    }

    private static void errorMessage(string msg)
    {
        Error( "Error: " + msg);
    }
    
    private static int convertToInt(string s)
    {
        int res = 0;
        for (int i = 0; i < s.Length; i++)
        {
            res = 10*res + (s[i]-'0');
        }
        return res;
    }

    private static string extractString(string profile, int index)
    {
        // 0 based index
        string res = "";
        int curIndex = 0;
        for (int i = 0; ; i++)
        {
            if (i >= profile.Length)
            {
                if (curIndex != index)
                {
                    res = "";
                }
                break;
            }
            if (profile[i] == '|')
            {
                if (curIndex == index)
                {
                    // Found it
                    break;
                }
                // Start new string
                curIndex++;
                res = "";
            }
            else
            {
                // Accumulate
                res += profile[i];
            }
        }
        return res;
    }
}
    /* Notes:
         CheckWitness
        * In many, if not all, cases, you’ll want to validate whether the address invoking your contract code is really who it says it is.
        * The Runtime.CheckWitness method accepts a single parameter that represents the address you’d like to validate against the address used to
        * invoke the contract code. More specifically, it verifies that the transactions or block of the calling contract has validated the
        * required script hashes.

        THrowing exceptions: generate VM Fault
        */

}