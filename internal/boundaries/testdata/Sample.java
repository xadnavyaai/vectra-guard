public class Sample {
    public native int nativeAdd(int a, int b);

    public int safeAdd(int a, int b) {
        return a + b;
    }
}
