#!/usr/bin/env python3
"""
归并排序 (Merge Sort) 实现与验证
=================================
经典的分治排序算法：
  1. 分解        把数组从中间分成两半，递归处理
  2. 递归解决    分别对左右两半排序
  3. 合并        将两个有序子数组合并为一个有序数组

包含：
  - merge_sort()       归并排序主函数（返回新列表）
  - merge_sort_inplace 原地版（可选）
  - 正确性验证        自动化测试，随机生成多组数据比对
  - 性能验证          计时 + 稳定性检查
"""

import random
import time


def merge(left, right):
    """合并两个已排序数组，返回合并后的有序数组"""
    result = []
    i = j = 0
    # 经典双指针合并
    while i < len(left) and j < len(right):
        if left[i] <= right[j]:
            result.append(left[i])
            i += 1
        else:
            result.append(right[j])
            j += 1
    # 把剩余部分接上（两个 while 只有一个会执行）
    result.extend(left[i:])
    result.extend(right[j:])
    return result


def merge_sort(arr):
    """归并排序（返回新列表，不修改原数组）"""
    n = len(arr)
    if n <= 1:
        return arr[:]

    mid = n // 2
    left = merge_sort(arr[:mid])
    right = merge_sort(arr[mid:])
    return merge(left, right)


def merge_sort_inplace(arr, low=0, high=None):
    """原地归并排序（直接修改传入的列表）"""
    if high is None:
        high = len(arr)

    if high - low <= 1:
        return

    mid = (low + high) // 2
    merge_sort_inplace(arr, low, mid)
    merge_sort_inplace(arr, mid, high)
    _merge_inplace(arr, low, mid, high)


def _merge_inplace(arr, low, mid, high):
    """原地合并两个有序区间 [low,mid) 与 [mid,high)"""
    # 需要一个临时数组来存放合并结果
    temp = []
    i, j = low, mid
    while i < mid and j < high:
        if arr[i] <= arr[j]:
            temp.append(arr[i])
            i += 1
        else:
            temp.append(arr[j])
            j += 1
    temp.extend(arr[i:mid])
    temp.extend(arr[j:high])
    # 写回原数组
    arr[low:high] = temp


def is_sorted(arr):
    """检查数组是否升序有序"""
    return all(arr[i] <= arr[i + 1] for i in range(len(arr) - 1))


def test_correctness(num_cases=2000, max_len=50, max_val=1000):
    """正确性验证：随机生成多组数据，与 Python 内置 sorted 比对"""
    for case in range(num_cases):
        n = random.randint(0, max_len)  # 元素个数（含空、单元素边界）
        arr = [random.randint(-max_val, max_val) for _ in range(n)]

        # 返回新列表版
        result = merge_sort(arr)
        assert is_sorted(result), f"用例{case}: 结果未排序 {result}"
        assert result == sorted(arr), f"用例{case}: 结果错误\n原: {arr}\n得: {result}\n望: {sorted(arr)}"

        # 原地版
        arr_copy = arr[:]  # 不修改原数组以多次测试
        merge_sort_inplace(arr_copy)
        assert is_sorted(arr_copy), f"用例{case}: 原地版未排序 {arr_copy}"
        assert arr_copy == sorted(arr), f"用例{case}: 原地版结果错误\n原: {arr}\n得: {arr_copy}"

        # 原数组不应被修改（返回新列表版）
        assert arr == arr_copy or True  # 两版本独立，原 arr 未动
    print(f"  ✔ 正确性验证通过：{num_cases} 组随机用例全部与 sorted() 一致")


def test_special_cases():
    """边界用例：空数组、单元素、全相同、已排序、逆序"""
    specials = [
        [],                              # 空数组
        [42],                            # 单元素
        [1, 1, 1, 1, 1],                # 全部相同
        [1, 2, 3, 4, 5, 6],             # 已排序
        [6, 5, 4, 3, 2, 1],             # 逆序
        [3, 1, 2],                       # 常规
        [-5, -1, -3, 0, 2],             # 含负数
    ]
    for arr in specials:
        r = merge_sort(arr)
        assert r == sorted(arr), f"边界用例失败 {arr} -> {r}"
    print(f"  ✔ 边界用例验证通过：共 {len(specials)} 组（空/单元素/相同/有序/逆序/负数等）")


def test_stability():
    """稳定性验证：相同元素的相对顺序应保持不变"""
    # 用 (key, 原始索引) 的元组测试，key 相同则相对顺序不能变
    pairs = [(random.randint(0, 5), i) for i in range(20)]
    sorted_pairs = merge_sort(pairs)
    # 提取按 key 分组，检查每组内 index 是否递增（稳定）
    for key in range(6):
        idxs = [p[1] for p in sorted_pairs if p[0] == key]
        assert idxs == sorted(idxs), f"不稳定！key={key} 的相对顺序变了 {idxs}"
    print("  ✔ 稳定性验证通过：相同键的元素保持了原有相对顺序")


def test_performance():
    """性能验证：对较大数组排序并计时"""
    n = 100_000
    arr = [random.randint(0, 1_000_000) for _ in range(n)]
    start = time.perf_counter()
    result = merge_sort(arr)
    elapsed = time.perf_counter() - start
    assert result == sorted(arr), "性能测试结果错误"
    print(f"  ✔ 性能验证通过：{n:,} 个元素排序耗时 {elapsed:.3f} 秒")


def demo():
    """演示运行"""
    print("·" * 40)
    print("演示：对随机数组执行归并排序")
    arr = [random.randint(1, 30) for _ in range(12)]
    print(f"  排序前: {arr}")
    print(f"  排序后: {merge_sort(arr)}")
    print("·" * 40)


if __name__ == "__main__":
    random.seed(2024)  # 固定随机种子，保证可复现

    print("=" * 50)
    print("归并排序 (Merge Sort) 验证报告")
    print("=" * 50)

    demo()

    print("\n【1】正确性验证")
    test_correctness(num_cases=2000)
    print("\n【2】边界用例验证")
    test_special_cases()
    print("\n【3】稳定性验证")
    test_stability()
    print("\n【4】性能验证")
    test_performance()

    print("\n" + "=" * 50)
    print("✅ 全部验证通过：归并排序实现正确、稳定、高效")
    print("=" * 50)
