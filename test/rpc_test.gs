package com.gsrpc.test;

using gslang.Package;
using gslang.Exception;

@Package(Lang:"golang",Name:"com.gsrpc.test",Redirect:"github.com/gsrpc/gorpc/test")

@Exception
table ResourceError {

}


contract RESTfull {
    string Get(string path) throws(ResourceError);
    void Put(string path,string val);
}
