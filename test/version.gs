package com.gsrpc.test;

using gslang.Package;

@Package(Lang:"golang",Name:"com.gsrpc.test",Redirect:"github.com/gsrpc/gorpc/test")


table V0 {
    int32   F1;
    float32 F2;
    byte[]  F3;
}

table V1 {
    int32       F1;
    float32     F2;
    byte[]      F3;
    string[]    F4;
    string[4]   F5;
}

table C0 {
    int32       C1;
    V0          C2;
    float32     C3;
}

table C1 {
    int32       C1;
    V1          C2;
    float32     C3;
    float64     C4;
}

@gslang.POD
table Hello {
    int32       C1;
    string[]    C2;
}
