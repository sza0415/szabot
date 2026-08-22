"""常用排序算法实现与测试"""

import heapq


def bubble_sort(arr):
    """冒泡排序：相邻比较，大的往后冒"""
    arr = arr[:]
    n = len(arr)
    for i in range(n):
        swapped = False
        for j in range(n - 1 - i):
            if arr[j] > arr[j + 1]:
                arr[j], arr[j + 1] = arr[j + 1], arr[j]
                swapped = True
        if not swapped:
            break
    return arr


def selection_sort(arr):
    """选择排序：每轮选出最小值放到前面"""
    arr = arr[:]
    n = len(arr)
    for i in range(n - 1):
        min_idx = i
        for j in range(i + 1, n):
            if arr[j] < arr[min_idx]:
                min_idx = j
        arr[i], arr[min_idx] = arr[min_idx], arr[i]
    return arr


def insertion_sort(arr):
    """插入排序：把每个元素插入到已排序部分的合适位置"""
    arr = arr[:]
    for i in range(1, len(arr)):
        key = arr[i]
        j = i - 1
        while j >= 0 and arr[j] > key:
            arr[j + 1] = arr[j]
            j -= 1
        arr[j + 1] = key
    return arr


def quick_sort(arr):
    """快速排序：分治，选基准，小的放左、大的放右"""
    if len(arr) <= 1:
        return arr
    pivot = arr[len(arr) // 2]
    left = [x for x in arr if x < pivot]
    middle = [x for x in arr if x == pivot]
    right = [x for x in arr if x > pivot]
    return quick_sort(left) + middle + quick_sort(right)


def merge_sort(arr):
    """归并排序：分治，递归拆分后合并有序子序列"""
    if len(arr) <= 1:
        return arr
    mid = len(arr) // 2
    left = merge_sort(arr[:mid])
    right = merge_sort(arr[mid:])
    result = []
    i = j = 0
    while i < len(left) and j < len(right):
        if left[i] <= right[j]:
            result.append(left[i])
            i += 1
        else:
            result.append(right[j])
            j += 1
    result.extend(left[i:])
    result.extend(right[j:])
    return result


def heap_sort(arr):
    """堆排序：构建小顶堆后依次弹出"""
    h = arr[:]
    heapq.heapify(h)
    return [heapq.heappop(h) for _ in range(len(h))]


def main():
    test_cases = [
        [5, 2, 8, 1, 9, 3],
        [3, 3, 1, 2, 1],
        [42],
        [],
        [10, 9, 8, 7, 6, 5, 4, 3, 2, 1],
    ]

    algorithms = {
        "冒泡排序": bubble_sort,
        "选择排序": selection_sort,
        "插入排序": insertion_sort,
        "快速排序": quick_sort,
        "归并排序": merge_sort,
        "堆排序": heap_sort,
    }

    for name, fn in algorithms.items():
        ok = True
        for case in test_cases:
            result = fn(case)
            expected = sorted(case)
            if result != expected:
                ok = False
                print(f"[FAIL] {name}  输入={case}  期望={expected}  实际={result}")
        print(f"[{'PASS' if ok else 'FAIL'}] {name}")

    # 演示一次
    demo = [5, 2, 8, 1, 9, 3]
    print(f"\n示例输入: {demo}")
    for name, fn in algorithms.items():
        print(f"  {name}: {fn(demo)}")


if __name__ == "__main__":
    main()
